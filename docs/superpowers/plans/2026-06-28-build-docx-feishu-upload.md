# build_docx → 飞书 Drive 上传 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `build_docx` 工具的输出从本地 `public/推介材料/<id>.docx` 改为上传到飞书 Drive：基础文件夹 `W9OGfnjzQl8dOOdqPFwcL6gEnkf` 下当年当月子文件夹（命名如「2026年6月」，不存在则新建），上传 .docx，返回飞书 URL。不在本地留存。

**Architecture:** 复用 feishu client 的 user-token OAuth（token 持久化在 `.feishu-user-token`，本地 + ECS 均已 OAuth）。给 feishu client 加 3 个方法（CreateFolder / FindSubfolder / UploadDocx，调飞书 Drive API），改 `build_docx`：写临时文件 → find-or-create 年月子文件夹 → 上传 → 删临时文件 → 返回飞书 URL。飞书可达 + token 有效，故可 live e2e。

**Tech Stack:** Go stdlib（`net/http`、`mime/multipart`、`bytes`、`encoding/json`、`os`、`time`）+ 已有 feishu client。

## Global Constraints

- **仅飞书，不落本地：** build_docx 写临时文件 → 上传 → 删临时文件，返回飞书 URL。上传失败 → `{"error":...}`，agent 告知用户，**不**回退到本地。
- **文件夹命名：** `time.Now().Format("2006年1月")` → 如「2026年6月」（月不补零）。
- **基础文件夹：** `W9OGfnjzQl8dOOdqPFwcL6gEnkf`，经 config `FeishuPitchFolderToken`（默认值即此，可 env 覆盖）。
- **飞书域名：** `kcngap16uccc.feishu.cn`（用户的租户域），经 config `FeishuDriveDomain`（默认即此）。文件 URL = `https://{domain}/file/{file_token}`。
- **凭证：** 复用 feishu client user-token（OAuth），token 文件 `.feishu-user-token`（与 Server 的 feishu client 共享）。不引新凭证。
- **upload_all 上限 20MB：** docx 当前 ~3KB，远低于。若未来超 20MB 需改分片上传（out of scope）。
- **可测性：** feishu client 新方法用包级 `feishuBase` 变量（默认 `https://open.feishu.cn`），httptest 可覆盖。`DriveFiles` 改 1 行用 `feishuBase`（保默认行为，使 FindSubfolder 可 mock）。
- **测试：** Task 1 feishu 方法用 httptest 单测（请求构造 + 响应解析，不碰真飞书）；Task 2 build_docx live e2e（真飞书上传，产一张真 docx 到年月文件夹——即功能正常工作的证据）。

---

## File Structure

- `backend-go/internal/feishu/client.go` — Task 1：加 `feishuBase` 包级变量、`CreateFolder`、`FindSubfolder`、`UploadDocx`、`postMultipart` helper；`DriveFiles` 1 行改用 `feishuBase`。
- `backend-go/internal/feishu/client_test.go` — Task 1：httptest 单测（CreateFolder/FindSubfolder/UploadDocx 请求+响应）。
- `backend-go/internal/config/config.go` — Task 2：加 `FeishuPitchFolderToken` + `FeishuDriveDomain`。
- `backend-go/internal/agent/service.go` — Task 2：改 `buildDocxTool`（临时文件→上传→删→飞书 URL）+ `systemPrompt` Word 步骤句。

---

### Task 1: feishu client — CreateFolder + FindSubfolder + UploadDocx + 单测

**Files:**
- Modify: `backend-go/internal/feishu/client.go`
- Test: `backend-go/internal/feishu/client_test.go`（新建）

**Interfaces:**
- Consumes: `ensureValidToken`、`post`、`getData`（已有）、`DriveFile`（已有）、`mime/multipart`、`bytes`、`encoding/json`、`net/http`、`fmt`、`strings`。
- Produces: `func (c *Client) CreateFolder(ctx, parentToken, name string) (folderToken string, err error)`、`func (c *Client) FindSubfolder(ctx, parentToken, name string) (folderToken string, found bool, err error)`、`func (c *Client) UploadDocx(ctx, parentFolderToken, fileName string, data []byte) (fileToken, url string, err error)`、`func (c *Client) postMultipart(ctx, endpoint string, fields map[string]string, fileName string, data []byte, bearer string) ([]byte, error)`、包级 `var feishuBase = "https://open.feishu.cn"`。Task 2 的 buildDocxTool 调用它们。

- [ ] **Step 1: 写失败测试**

`backend-go/internal/feishu/client_test.go`:
```go
package feishu

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newAuthedClient 建一个带未过期 user token 的 client（ensureValidToken 不触发刷新）。
func newAuthedClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c := New("app", "secret", "http://redirect")
	c.HTTP = srv.Client()
	c.userToken = "test-token"
	c.expiresAt = getTimeFuture() // 远未来，确保不刷新
	return c
}

func TestCreateFolder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/drive/v1/files/create_folder" {
			t.Errorf("path = %s, want create_folder", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth = %q, want Bearer test-token", got)
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"name":"2026年6月"`) || !strings.Contains(string(b), `"folder_token":"parentT"`) {
			t.Errorf("body = %s, want name+folder_token", b)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"token":"fldNew","url":"https://x.feishu.cn/drive/folder/fldNew"}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	tok, err := c.CreateFolder(context.Background(), "parentT", "2026年6月")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if tok != "fldNew" {
		t.Errorf("token = %q, want fldNew", tok)
	}
}

func TestFindSubfolder_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/open-apis/drive/v1/files") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"files":[
			{"token":"fldA","name":"2026年5月","type":"folder"},
			{"token":"fldB","name":"2026年6月","type":"folder"},
			{"token":"filC","name":"some.docx","type":"docx"}
		],"has_more":false}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	tok, found, err := c.FindSubfolder(context.Background(), "parentT", "2026年6月")
	if err != nil {
		t.Fatalf("FindSubfolder: %v", err)
	}
	if !found || tok != "fldB" {
		t.Errorf("got tok=%q found=%v, want fldB/true", tok, found)
	}
}

func TestFindSubfolder_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"files":[{"token":"fldA","name":"2026年5月","type":"folder"}],"has_more":false}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	tok, found, err := c.FindSubfolder(context.Background(), "parentT", "2026年6月")
	if err != nil || found {
		t.Fatalf("got tok=%q found=%v err=%v, want false/nil", tok, found, err)
	}
}

func TestUploadDocx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/drive/v1/files/upload_all" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth = %q", got)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("content-type = %q, want multipart", ct)
		}
		b, _ := io.ReadAll(r.Body)
		body := string(b)
		for _, want := range []string{`name="file_name"`, "pitch.docx", `name="parent_type"`, "explorer", `name="parent_node"`, "fldMonth", `name="size"`, `filename="pitch.docx"`} {
			if !strings.Contains(body, want) {
				t.Errorf("multipart body missing %q", want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"data":{"file_token":"filNew"}}`))
	}))
	t.Cleanup(srv.Close)
	orig := feishuBase
	feishuBase = srv.URL
	t.Cleanup(func() { feishuBase = orig })

	c := newAuthedClient(t, srv)
	ft, url, err := c.UploadDocx(context.Background(), "fldMonth", "pitch.docx", []byte("fake docx bytes"))
	if err != nil {
		t.Fatalf("UploadDocx: %v", err)
	}
	if ft != "filNew" {
		t.Errorf("file_token = %q, want filNew", ft)
	}
	if !strings.HasSuffix(url, "/file/filNew") {
		t.Errorf("url = %q, want suffix /file/filNew", url)
	}
}
```

**注意：** 测试用 `getTimeFuture()` 与 `c.userToken`/`c.expiresAt` 直接字段访问——这些在 `client.go` 已是可访问的包内字段（`userToken`/`expiresAt` 是小写字段，同包可访问）。`getTimeFuture()` 需在测试文件或 client.go 提供；为避免引入 `time.Now()`（workflow 限制不适用，这是 Go 测试），在测试文件加 `func getTimeFuture() time.Time { return time.Date(2099,1,1,0,0,0,0,time.UTC) }` 并 import `time`。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/feishu/ -v`
Expected: FAIL with `undefined: CreateFolder` / `feishuBase` / `postMultipart`。

- [ ] **Step 3: 写实现**

`backend-go/internal/feishu/client.go`：

1. 顶部加包级变量（在 `var` 区或 `New` 之前）：
```go
// feishuBase 是飞书开放平台 base URL，默认官方；测试可覆盖以指向 httptest。
var feishuBase = "https://open.feishu.cn"
```

2. `DriveFiles` 里把硬编码 `https://open.feishu.cn` 改为 `feishuBase`（1 行）：
```go
return c.getData(ctx, feishuBase+"/open-apis/drive/v1/files?"+params.Encode())
```

3. 文件末尾追加 3 个方法 + multipart helper：
```go
// CreateFolder 在 parentToken 下创建名为 name 的文件夹，返回新文件夹 token。
func (c *Client) CreateFolder(ctx context.Context, parentToken, name string) (string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return "", err
	}
	body, err := c.post(ctx, feishuBase+"/open-apis/drive/v1/files/create_folder", map[string]any{
		"name":         name,
		"folder_token": parentToken,
	}, token)
	if err != nil {
		return "", err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("create_folder parse: %w", err)
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("create_folder code %d: %s", resp.Code, resp.Msg)
	}
	if resp.Data.Token == "" {
		return "", fmt.Errorf("create_folder returned empty token")
	}
	return resp.Data.Token, nil
}

// FindSubfolder 在 parentToken 下找名为 name 的子文件夹，返回其 token + 是否找到。
func (c *Client) FindSubfolder(ctx context.Context, parentToken, name string) (string, bool, error) {
	raw, err := c.DriveFiles(ctx, parentToken, "", "")
	if err != nil {
		return "", false, err
	}
	data, _ := raw["data"].(map[string]any)
	if data == nil {
		return "", false, nil
	}
	files, _ := data["files"].([]any)
	for _, f := range files {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		n, _ := m["name"].(string)
		ty, _ := m["type"].(string)
		if n == name && (ty == "folder" || ty == "") {
			if tok, _ := m["token"].(string); tok != "" {
				return tok, true, nil
			}
		}
	}
	return "", false, nil
}

// UploadDocx 把 data 作为 fileName 上传到 parentFolderToken，返回 file_token + 飞书 URL。
// 用 upload_all（单次，< 20MB）。
func (c *Client) UploadDocx(ctx context.Context, parentFolderToken, fileName string, data []byte) (string, string, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return "", "", err
	}
	body, err := c.postMultipart(ctx, feishuBase+"/open-apis/drive/v1/files/upload_all",
		map[string]string{
			"file_name":   fileName,
			"parent_type": "explorer",
			"parent_node": parentFolderToken,
			"size":        fmt.Sprintf("%d", len(data)),
		}, fileName, data, token)
	if err != nil {
		return "", "", err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			FileToken string `json:"file_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", "", fmt.Errorf("upload_all parse: %w", err)
	}
	if resp.Code != 0 {
		return "", "", fmt.Errorf("upload_all code %d: %s", resp.Code, resp.Msg)
	}
	if resp.Data.FileToken == "" {
		return "", "", fmt.Errorf("upload_all returned empty file_token")
	}
	return resp.Data.FileToken, feishuDriveDomain + "/file/" + resp.Data.FileToken, nil
}

// postMultipart 发 multipart/form-data 请求。fields 为普通字段，fileName+data 为文件字段。
func (c *Client) postMultipart(ctx context.Context, endpoint string, fields map[string]string, fileName string, data []byte, bearer string) ([]byte, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	fw, err := mw.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	if _, err := out.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feishu API status %d: %s", resp.StatusCode, out.String())
	}
	return out.Bytes(), nil
}
```
import 区补 `"mime/multipart"`、`"bytes"`（若缺；`net/http`/`encoding/json`/`fmt` 已有）。

**注意 `feishuDriveDomain`：** UploadDocx 引用 `feishuDriveDomain`——这是 Task 2 在 config 包加的。但 feishu 包不应直接依赖 config（避免循环）。改为：UploadDocx 只返回 file_token，URL 由调用方（buildDocxTool，Task 2）用 `s.cfg.FeishuDriveDomain` 构造。**修正：** UploadDocx 签名改为 `(fileToken string, err error)`（不返回 url），测试相应去掉 url 断言。Task 2 的 buildDocxTool 用 `https://`+`s.cfg.FeishuDriveDomain`+`/file/`+fileToken 构造 URL。

→ **Step 3 修正版：** UploadDocx 返回 `(fileToken string, err error)`；删除 `feishuDriveDomain` 引用。测试 `TestUploadDocx` 改为断言 `ft == "filNew"`（去掉 url 断言）。

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/feishu/ -v`
Expected: PASS（4 个子测试）。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/feishu/client.go backend-go/internal/feishu/client_test.go
git commit -m "feat(feishu): add CreateFolder/FindSubfolder/UploadDocx (Drive API)"
```

---

### Task 2: config + build_docx 改飞书上传 + systemPrompt + live e2e

**Files:**
- Modify: `backend-go/internal/config/config.go`
- Modify: `backend-go/internal/agent/service.go`（`buildDocxTool` + `systemPrompt`）

**Interfaces:**
- Consumes: `feishu.New`、`SetTokenPersistPath`、`FindSubfolder`、`CreateFolder`、`UploadDocx`（Task 1）、`BuildDocx`（S3）、`stringArg`、`os`、`time`、`fmt`、`config.Config`。
- Produces: build_docx 上传飞书返回飞书 URL；config 加 2 字段。

- [ ] **Step 1: config 加 2 字段**

`backend-go/internal/config/config.go`，Config struct 加：
```go
	FeishuPitchFolderToken string
	FeishuDriveDomain      string
```
Load() return 字面量加：
```go
		FeishuPitchFolderToken: getEnv("FEISHU_PITCH_FOLDER_TOKEN", "W9OGfnjzQl8dOOdqPFwcL6gEnkf"),
		FeishuDriveDomain:      getEnv("FEISHU_DRIVE_DOMAIN", "kcngap16uccc.feishu.cn"),
```

- [ ] **Step 2: 改 `buildDocxTool`**

`backend-go/internal/agent/service.go` 的 `buildDocxTool`：把「写 public/推介材料 + 返回 /public URL」改为「写临时文件 → 上传飞书 → 删临时 → 返回飞书 URL」。替换整个方法体：
```go
// buildDocxTool 是 agent 工具入口：装配 .docx 并上传到飞书 Drive 当年当月子文件夹，返回飞书 URL。
func (s *Service) buildDocxTool(args map[string]any) map[string]any {
	sections, err := parseManifest(args)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	// 1. 写临时文件
	tmpDir, err := os.MkdirTemp("", "docx-")
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, "推介材料.docx")
	if err := BuildDocx(sections, tmpPath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	// 2. 飞书 client（复用持久化 user token）
	fc := feishu.New(s.cfg.FeishuAppID, s.cfg.FeishuAppSecret, s.cfg.FeishuRedirectURI)
	fc.SetTokenPersistPath(".feishu-user-token")
	// 3. find-or-create 当年当月子文件夹
	folderName := time.Now().Format("2006年1月")
	ctx := context.Background()
	subToken, found, err := fc.FindSubfolder(ctx, s.cfg.FeishuPitchFolderToken, folderName)
	if err != nil {
		return map[string]any{"error": "查找飞书文件夹失败：" + err.Error()}
	}
	if !found {
		subToken, err = fc.CreateFolder(ctx, s.cfg.FeishuPitchFolderToken, folderName)
		if err != nil {
			return map[string]any{"error": "创建飞书文件夹失败：" + err.Error()}
		}
	}
	// 4. 上传
	fileName := fmt.Sprintf("推介材料_%s.docx", time.Now().Format("20060102_150405"))
	fileToken, err := fc.UploadDocx(ctx, subToken, fileName, data)
	if err != nil {
		return map[string]any{"error": "上传飞书失败：" + err.Error()}
	}
	url := "https://" + s.cfg.FeishuDriveDomain + "/file/" + fileToken
	return map[string]any{"url": url, "file_token": fileToken, "folder": folderName}
}
```
import 区补 `"context"`、`"path/filepath"`、`"business-workbench/backend-go/internal/feishu"`（若缺）。`time`/`os`/`fmt` 已有。

- [ ] **Step 3: systemPrompt Word 步骤句更新**

`service.go` `systemPrompt` 里把 S3 写的「...调 build_docx」段中关于输出位置的描述更新：把「build_docx 装配 .docx」后追加「（上传到飞书 Drive 当年当月文件夹，返回飞书 URL）」。具体：在「用户要出 Word 材料时：按 references/docx-template.md 的 10 章顺序组装 manifest 调 build_docx。」这句的「build_docx」后插「（上传飞书 Drive，返回飞书链接，不落本地）」。

- [ ] **Step 4: 编译 + 全测试**

Run: `cd backend-go && go build ./... && go test ./internal/feishu/ ./internal/agent/ -v`
Expected: 编译通过；feishu 4 测试 + agent 测试全 PASS。

- [ ] **Step 5: live e2e（真飞书上传，本环境可跑——token 已持久化 + 飞书可达）**

启动后端（`WINRATE_DRY_RUN=true` 可选，使 fetch_winrate 不卡），触发一次要求出 Word 的对话（用 S3 e2e 同款参数 + 历史底部 + 胜率）。Expected：
- SSE 出现 `tool_call: build_docx` → `tool_done`，结果含 `{"url":"https://kcngap16uccc.feishu.cn/file/...","folder":"2026年X月"}`。
- 打开该飞书 URL：能看到下载的 .docx；打开 .docx 内容正常（长版/短版文案、参数块、缺图红字占位）。
- 飞书 Drive 基础文件夹下出现当年当月子文件夹（若本月首次），里面有该 .docx。
- 无 `public/推介材料/` 本地文件残留（临时文件已删）。

若上传失败（如 token 过期且刷新失败）：build_docx 返回 `{"error":"上传飞书失败：..."}`，agent 告知用户——不回退本地。

- [ ] **Step 6: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/config/config.go backend-go/internal/agent/service.go
git commit -m "feat(agent): build_docx uploads to Feishu Drive year-month folder"
```

---

## Self-Review

**1. Spec coverage:**
- 不落本地（临时文件→上传→删）→ Task 2 buildDocxTool。✅
- 当年当月子文件夹（2026年6月）→ Task 2 time.Now().Format("2006年1月")。✅
- find-or-create → Task 2 FindSubfolder + CreateFolder。✅
- 基础文件夹 token → config FeishuPitchFolderToken（默认 W9OG...）。✅
- 飞书 URL → Task 2 构造 https://{domain}/file/{token}。✅
- 飞书 client 3 方法 → Task 1。✅
- 复用 user token → Task 2 SetTokenPersistPath(".feishu-user-token")。✅
- 上传失败 → error，不回退本地 → Task 2 各步 error 返回。✅
- 测试：httptest 单测（Task 1）+ live e2e（Task 2）。✅

**2. Placeholder scan:** 无 TBD/TODO。飞书 API endpoint/body/response 全具体。Task 1 Step 3 的 `feishuDriveDomain` 引用已在 Step 3 修正为 UploadDocx 只返 file_token（URL 由 Task 2 构造），避免 feishu 包依赖 config。测试 `getTimeFuture()` 显式定义。

**3. Type consistency:** `CreateFolder(ctx, parentToken, name) (string, error)`、`FindSubfolder(ctx, parentToken, name) (string, bool, error)`、`UploadDocx(ctx, parentFolderToken, fileName string, data []byte) (string, error)`（修正后只返 file_token）在 Task 1 定义、Task 2 调用一致。`postMultipart(ctx, endpoint, fields, fileName, data, bearer) ([]byte, error)` 内部用。`feishuBase` 包级变量 Task 1 定义、测试覆盖。config `FeishuPitchFolderToken`/`FeishuDriveDomain` Task 2 定义、buildDocxTool 调用一致。`BuildDocx(sections, outputPath)`（S3）Task 2 调用一致。`parseManifest(args)`（S3）Task 2 调用一致。
