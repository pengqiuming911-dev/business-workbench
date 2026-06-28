# 产品点位卡 + PDF 生成流程

本文档说明如何用通毓终端的"产品点位"小工具(`terminal.tongyu-quant.com/smallTool/index.html#/product-position`)根据产品参数生成一张**结构解析卡**(含敲出参考图),并把这张卡作为图片嵌入最终的推介 PDF。

## 何时用

文案写完、胜率取到后,如果用户要"出 Word 材料",走本流程:把产品数据填进产品点位页 → 生成卡 → 取图 → 装进 Word。用户没要出材料时,只出长版+短版文字文案即可,不强制跑这步。

## 进入页面

登录终端后(登录见 `tongyu-winrate.md`),导航到 `https://terminal.tongyu-quant.com/smallTool/index.html#/product-position`(标题"产品点位 - 金融产品工具箱")。与主站共享登录态,一般不用重新登录。

## 填表:产品参数 → 表单字段

**先选产品类型再填数**——表单字段随产品类型动态变化(DCN 和锁盈的票息字段不同)。映射:

| 产品参数 | 表单字段 | 说明 |
|---|---|---|
| — | 产品名称 | 自定义,如"中证1000 2倍DCN" |
| 结构类型 | 产品类型 | 原生 select:DCN / 锁盈 |
| 期限 | 期限(月) | 如 36 |
| 锁定期 | 锁定期(月) | 如 3 |
| 保证金 | 保证金(%) | 如 50 |
| 期初敲出线 | 敲出线(%) | 如 101 |
| 降敲 | 降敲(每月) | 如 0.5 |
| 降落伞 | 降落伞(%) | 如 60 |
| — | 降落伞月份 | 降落伞开始观察的月份,默认 1 |
| 费后派息 | 每月或有派息(%) | **DCN 下出现**,直接填月票息,如 1.39 |
| 派息线 | 派息线(%) | **DCN 下出现**,如 78 |
| 入场时间 | 入场时间(日期) | 如 2026-07-03 |
| 入场点位 | 入场点位 | 指数点位,用 `fetch_quote` 工具取的当前点位,或入场日预估 |

**票息口径坑**:DCN 用"每月或有派息"(月票息,直接填 1.39);锁盈/经典结构用"区间年化票息"(年化,月票息×12,如 1.39%/月 → 填 16.68)。表单 placeholder 会提示"区间年化票息(%)"或"每月派息比例",按提示口径填,别填错单位。

填数技巧:`screenshot_product_card` 工具(chromedp)内部按上面的字段映射填表(antd InputNumber 用 SetValue/SendKeys,产品类型原生 `<select>` 用 JS 设 selectedIndex)。你只需把产品参数传给工具,不用自己驱动浏览器。表单字段随产品类型动态变化,执行时每次操作后重新快照、按字段名/placeholder 定位当前 ref,别记死 ref。

## 生成卡 + 取图

调 `screenshot_product_card` 工具(chromedp):工具会自动点"提交"→接受 alert→点"📋 复制为图片"→优先读剪贴板 PNG,兜底对结果卡元素(region,class 含 `product-result`)截图。返回 PNG 的 `/public/...` path + URL。你只需传产品参数,无需手动处理 alert/剪贴板/截图。

## 装进 Word

产品卡 PNG 是 Word 推介材料"产品派息与敲出观察点位表"那一节的图。最终装配用 `build_docx` 工具 + manifest(见 `references/docx-template.md`),不再用 PDF。manifest 里这一节写:

```json
{"type":"heading","text":"产品派息与敲出观察点位表"},
{"type":"image","path":"product-card.png","caption":"产品派息与敲出观察点位表(通毓终端生成)"}
```

build_docx 对偏高图片自动按 20cm 高度上限缩放,不用手动改图。

## 字段会变怎么办

同 `tongyu-winrate.md`:小工具也是动态表单,ref 每次交互后会变。本文档给语义映射,执行时每次操作后重新快照、按字段名/placeholder 定位当前 ref,别记死 ref。
