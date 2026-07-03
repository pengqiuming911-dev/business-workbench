# 飞书 .docx 二进制上传接口 + skill 集成

日期: 2026-07-03
状态: 设计已确认，待实现

## 目标

让 `structured-product-copywriter` skill 能把含截图的完整推介材料(.docx)上传到飞书云空间「当年当月」子文件夹，返回飞书 `/file/` 链接。复用 workbench 已有飞书 OAuth token 与上传逻辑，不要求本地常驻 Go 服务。

## 背景

workbench 已开放两个飞书接口：
- `POST /api/drive/build-docx`（已上生产）：接收 sections JSON，服务端装配 .docx 并 UploadDocx。
- `POST /api/drive/create-docx`（仅 feature 分支 `fix/preserve-internal-docx-token`）：创建原生云文档，写 markdown。

`build-docx` 的 image section 用 `os.ReadFile(s.Path)` 读**服务端本地文件系统** PNG（`docx.go:149-154`），不支持 URL/base64。而推介材料截图（通毓回测、AMAC 公示、产品卡）在用户本机由 Playwright 生成——生产服务器读不到。直接调生产 `build-docx` 会丢图（变红字占位）。

本地调 `build-docx` 可行，但要求本地常驻 Go 服务。改为：本地 `build_docx.py` 装配好 .docx（图片本地可读），把 .docx 二进制 POST 到新增的生产接口，生产只负责 UploadDocx。

## workbench 后端改动（`D:\projects\business-workbench\backend-go`）

### 新接口 `POST /api/drive/upload-docx`

- 路由：`router.go` 新增 `router.POST("/api/drive/upload-docx", server.driveUploadDocx)`
- handler `driveUploadDocx`（仿 `driveBuildDocx`，`router.go:618-639`）：
  - 鉴权：`X-Internal-Token`(env `INTERNAL_DOCX_TOKEN`，空则跳过) + `requireFeishuAccess`
  - 请求体 `multipart/form-data`：
    - `file`（必填）：.docx 二进制
    - `title`（可选）：文件名；缺省 `推介材料_YYYYMMDD_HHMMSS.docx`
  - 逻辑：
    1. `c.FormFile("file")` 读二进制
    2. feishu client（`SetTokenPersistPath(".feishu-user-token")`）
    3. `yearMonth := time.Now().Format("2006年1月")`  // "2026年7月"
    4. `subToken, found := fc.FindSubfolderFuzzy(ctx, rootToken, yearMonth)`
    5. `!found → subToken = fc.CreateFolder(ctx, rootToken, yearMonth+"产品")`  // 新建沿用"产品"后缀，与既有产品材料文件夹一致
    6. `fileName := sanitizeFileName(title)+".docx"`（或默认）
    7. `fileToken := fc.UploadDocx(ctx, subToken, fileName, data)`
    8. 返回 `{url, file_token, folder}`
- `url = "https://"+FeishuDriveDomain+"/file/"+fileToken`

### `FindSubfolderFuzzy`（`client.go`，仿 `FindSubfolder` `client.go:891-912`）

- 列根文件夹下子文件（`DriveFiles`，page_size=50）
- 取 `type=folder` 且 `name` 包含 `yearMonth` 子串的第一个；多命中取第一个
- 都不命中返回 `(nil, false, nil)`
- 同时把 `buildDocxTool`（`service.go:610-621`）与 `driveCreateDocx`（`router.go:675-692`）的 `FindSubfolder` 改为 `FindSubfolderFuzzy`，保持三处一致；新建时用 `yearMonth+"产品"`

### 部署

生产 47.103.54.197，Go :3001，Nginx `proxy_pass http://localhost:3001`。在新分支或 `fix/preserve-internal-docx-token` 上实现，合并 main，走 `business-workbench-ops` 流程编译重启。

## skill 补充（`structured-product-copywriter`）

### SKILL.md 新增「第六步（可选）：上传飞书云文档」

触发：用户说"传飞书/归档飞书/上传飞书"。
流程：
1. 装配 manifest（第五步已产出；image path 用本地绝对路径；截图必须 PNG）
2. `build_docx.py` 装配 .docx（图片本地可读）
3. `python scripts/upload_to_feishu.py --manifest manifest.json --title "中证1000 2倍降敲DCN 2026-07-03"`
4. 打印飞书 `/file/` 链接

标题格式：标的+结构+日期（如 `中证1000 2倍降敲DCN 2026-07-03`）。

### `scripts/upload_to_feishu.py`

- 复用 `build_docx.build()` 装配 .docx 到临时文件
- 读二进制
- POST multipart(`file` + `title`) 到 `{BASE_URL}/api/drive/upload-docx`
- `BASE_URL` 默认 `http://47.103.54.197`，env `WORKBENCH_BASE_URL` 可覆盖
- header `X-Internal-Token`: env `INTERNAL_DOCX_TOKEN`（**不写进 skill 文件**）
- 打印返回 url；失败打印错误体

### `references/feishu-upload.md`

三接口契约表、图片本地约束（为什么走二进制接口）、模糊匹配逻辑、凭据/token 前置、生产 token 失效排查、本地调 fallback。

## 数据流

Playwright 截图（本地 PNG）→ `build_docx.py` 装配 .docx（图嵌入）→ `upload_to_feishu.py` POST 二进制 → 生产 `upload-docx` → UploadDocx 到「2026年7月产品」文件夹 → 返回 `/file/` 链接

## 错误处理

- 生产 token 失效：`requireFeishuAccess` 返回 401 → 脚本打印"飞书 token 失效，需重新 OAuth 授权（localhost:3001 回调，生产需 SSH 隧道或临时改 redirect_uri）"，退出
- 上传失败：打印服务端 error 体
- 图缺失：`build_docx.py` 已插红字占位，不阻断；.docx 仍上传

## 测试

- 后端：`FindSubfolderFuzzy` 单测（命中/不命中/多命中）；手测 curl 上传一个含图 .docx，验证飞书链接能打开且图在
- skill：跑 `upload_to_feishu.py` 对一份测试 manifest，确认返回链接

## 前置/风险

- 生产 `.feishu-user-token` 有效性：实现时先 curl 生产 `/api/drive/build-docx`（已部署）验证 token；失效则需重新授权
- 新接口不影响既有 `build-docx`/`create-docx`
- `INTERNAL_DOCX_TOKEN` 生产是否设置：若设置，脚本须带；值从 env 读

## 非目标（YAGNI）

- 不改 `build-docx` 支持图片 URL/base64（二进制接口已解耦）
- 不在本机常驻 Go 服务
- 不部署 `create-docx`（接口2）——本次用 .docx 文件上传即可
- 不做断点续传/大文件分片（推介材料 .docx 通常 < 5MB）
