# Copywriter Skill → Agent (S3 Word 推介材料) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 agent 加 Word 推介材料装配能力：`build_docx` 工具用一个 manifest（章节 + 内联文案 + 图片路径 + 超链接）装配出 `.docx`，经 `/public` 托管下载；`screenshot_amac` / `screenshot_product_card` 两个 chromedp 工具取 AMAC 公示图与通毓产品点位卡图作为图片来源。

**Architecture:** docx 装配用**自写最小纯 Go 写入器**（`archive/zip` + `encoding/xml`，无第三方 docx 库——unioffice 商业、fumiama AGPL，均不可用；用户已选自写）。支持 7 种 section：heading/subheading/body/params/image/separator/link_list；图片 PNG 解码后按 20cm 高度上限等比缩放嵌入；缺图插红字 `[图片待补:xxx]`；剥离 BMP 外 emoji；字体微软雅黑(eastAsia)、参数块 Consolas。截图工具复用 S2 的 `browser.go` chromedp 基建。manifest 由 agent 内联组装（文案已在对话上下文，不走 copy_file）。

**Tech Stack:** Go stdlib（`archive/zip`、`encoding/xml`、`image/png`、`os`、`path/filepath`）+ 已有 `chromedp`（S2 引入）。

## Global Constraints

- **docx 库：自写最小纯 Go 写入器**，**不引** unioffice（商业）/ fumiama（AGPL）/ 其他第三方 docx 库。仅 stdlib。
- **manifest 内联：** agent 把长版/短版文案文本、胜率文本、截图 path（来自 screenshot 工具或占位）直接写进 manifest 的 section，**不走 copy_file 读文件**。
- **图片缩放：** image section 的 PNG 解码取宽高，按 **20cm 高度上限**等比缩放（EMU: 914400/英寸、360000/cm；PNG 无物理 DPI，按 96px/inch 换算原始尺寸）。
- **缺图占位：** image section 的 path 读不到文件时，插红字 `[图片待补:<caption>]`，**不报错**（一页通/托管/胜率截图这类用户手动贴或 v1 未取的，留占位）。
- **emoji 剥离：** 文本入 docx 前剥离 BMP 外字符（🚀 等，雅黑不含），标题文字照常显示。
- **字体：** 正文微软雅黑（run 的 rPr 设 eastAsia）；params 等宽小字用 Consolas+雅黑。
- **数字零例外（沿用）：** manifest 里的文案数字必须与 fetch_quote/calc_points/fetch_winrate 的工具结果一致；build_docx 只装配，不改数字。
- **截图 live = 人工验收：** AMAC 与通毓产品卡 chromedp 流程按参考文档语义写，本环境无法访问 AMAC/通毓，live e2e 由用户执行；选择器首次 live 运行需调（参考文档预见）。docx 写入器纯 stdlib，可单测。
- **依赖前置：** Task 1（docx 文本 section，纯 stdlib，单测）；Task 2（docx image section，单测）；Task 3（chromedp 截图，manual-acceptance）；Task 4（注册 + prompt + e2e）。

---

## File Structure

- `backend-go/internal/agent/docx.go` — 自写 docx 写入器。Task 1 建文本 section（heading/subheading/body/params/separator/link_list）+ zip + emoji/字体 + image 占位；Task 2 把 image 占位换成真嵌入。
- `backend-go/internal/agent/docx_test.go` — `BuildDocx` 单测：固定 manifest → 校验 zip 内容 + XML 文本 + 图片嵌入 + 缺图占位。
- `backend-go/internal/agent/browser.go` — Task 3 加 `screenshotAMAC` + `screenshotProductCard`（复用 S2 的 `newBrowserContext`/tongyu 登录）。
- `backend-go/internal/agent/service.go` — Task 4：`executeTool` switch 加 3 case、`toolDefinitions` 加 3 项、`systemPrompt` 加 Word 步骤、3 个 `(s *Service)` 方法 + `parseManifest`/`nextPublicID`。

---

### Task 1: docx 写入器 — 文本 section + link_list + zip + emoji/字体

自写最小 docx 写入器的文本部分：heading/subheading/body/params/separator/link_list 六种 section + zip 装配 + emoji 剥离 + 字体。image section 本任务写红字占位（Task 2 换真嵌入）。纯 stdlib，TDD。

**Files:**
- Create: `backend-go/internal/agent/docx.go`
- Test: `backend-go/internal/agent/docx_test.go`

**Interfaces:**
- Consumes: `archive/zip`、`strings`、`unicode/utf8`、`os`、`path/filepath`、`fmt`。
- Produces: `type docxSection struct { Type, Text, Path, Caption string; Items []docxLink }`、`type docxLink struct { Label, URL string }`、`type docxBuilder struct { imgIdx, linkIdx int }`、`func BuildDocx(sections []docxSection, outputPath string) error`、`func writeZip(w *zip.Writer, name, content string) error`、`func imagePlaceholderXML(caption string) string`、`func stripNonBmp(s string) string`、`func xmlEscape(s string) string`。Task 2 扩展 image；Task 4 的 `(s *Service) buildDocxTool` 调用 `BuildDocx`。

- [ ] **Step 1: 写失败测试**

`backend-go/internal/agent/docx_test.go`:
```go
package agent

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDocx_TextSections(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "test.docx")
	sections := []docxSection{
		{Type: "heading", Text: "产品结构文字版（长版）"},
		{Type: "subheading", Text: "稳定版接龙通知"},
		{Type: "body", Text: "中证1000当前点位在8600点左右。\n安全垫厚。"},
		{Type: "params", Text: "标的：中证1000\n期限：36M"},
		{Type: "separator"},
		{Type: "link_list", Items: []docxLink{
			{Label: "管理人相关常见问题", URL: "https://example.com/docx/Qsi"},
			{Label: "交易台相关问题", URL: "https://example.com/docx/JVs"},
		}},
	}
	if err := BuildDocx(sections, out); err != nil {
		t.Fatalf("BuildDocx: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() == 0 {
		t.Fatalf("docx 未生成或为空: %v", err)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("zip open: %v", err)
	}
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/_rels/document.xml.rels"} {
		if !names[want] {
			t.Errorf("zip 缺 %s", want)
		}
	}

	doc := readZipFile(t, zr, "word/document.xml")
	for _, want := range []string{"产品结构文字版（长版）", "中证1000当前点位", "安全垫厚", "标的：中证1000", "管理人相关常见问题"} {
		if !strings.Contains(doc, want) {
			t.Errorf("document.xml 缺文本 %q", want)
		}
	}
	rels := readZipFile(t, zr, "word/_rels/document.xml.rels")
	if !strings.Contains(rels, "https://example.com/docx/Qsi") || !strings.Contains(rels, "https://example.com/docx/JVs") {
		t.Errorf("document.xml.rels 缺超链接 URL: %s", rels)
	}
}

func TestBuildDocx_StripsNonBmpEmoji(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "e.docx")
	// 🚀 是 BMP 外字符（U+1F680），应被剥离；标题文字保留
	if err := BuildDocx([]docxSection{{Type: "heading", Text: "🚀收益安全垫双重加厚"}}, out); err != nil {
		t.Fatalf("BuildDocx: %v", err)
	}
	zr, _ := zip.OpenReader(out)
	defer zr.Close()
	doc := readZipFile(t, zr, "word/document.xml")
	if strings.Contains(doc, "🚀") {
		t.Error("BMP 外 emoji 未被剥离")
	}
	if !strings.Contains(doc, "收益安全垫双重加厚") {
		t.Error("标题文字被误删")
	}
}

func readZipFile(t *testing.T, zr *zip.ReadCloser, name string) string {
	t.Helper()
	f, err := zr.Open(name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer f.Close()
	b, _ := io.ReadAll(f)
	return string(b)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/agent/ -run TestBuildDocx -v`
Expected: FAIL with `undefined: BuildDocx` / `undefined: docxSection`。

- [ ] **Step 3: 写实现（文本部分）**

`backend-go/internal/agent/docx.go`:
```go
package agent

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// docxSection 是推介材料 docx 的一个章节。Type 决定其余字段的含义。
// heading/subheading/body/params/separator 用 Text；image 用 Path+Caption（Task 2）；link_list 用 Items。
type docxSection struct {
	Type    string
	Text    string
	Path    string
	Caption string
	Items   []docxLink
}

type docxLink struct {
	Label string
	URL   string
}

// docxBuilder 在装配过程中计数关系 id（图片 Task 2 用 imgIdx、超链接用 linkIdx）。
type docxBuilder struct {
	imgIdx  int
	linkIdx int
}

// BuildDocx 把 sections 装配成 .docx 写到 outputPath。
// 文本 section：heading/subheading/body/params/separator/link_list（本任务）。
// image section：Task 1 写红字占位；Task 2 换成真嵌入（path 读不到仍占位）。
// 剥离 BMP 外 emoji；正文微软雅黑，params Consolas+雅黑。
func BuildDocx(sections []docxSection, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	b := &docxBuilder{}
	var body strings.Builder
	var rels strings.Builder
	for _, s := range sections {
		b.writeSection(&body, &rels, s)
	}

	if err := writeZip(zw, "[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Default Extension="png" ContentType="image/png"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`); err != nil {
		return err
	}
	if err := writeZip(zw, "_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`); err != nil {
		return err
	}
	if err := writeZip(zw, "word/document.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body>`+body.String()+`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr></w:body></w:document>`); err != nil {
		return err
	}
	if err := writeZip(zw, "word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+rels.String()+`</Relationships>`); err != nil {
		return err
	}
	return nil
}

func (b *docxBuilder) writeSection(body, rels *strings.Builder, s docxSection) {
	clean := stripNonBmp(s.Text)
	switch s.Type {
	case "heading":
		body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` + runText(clean, "微软雅黑", "") + `</w:p>`)
	case "subheading":
		body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading2"/></w:pPr>` + runText(clean, "微软雅黑", "") + `</w:p>`)
	case "body":
		for _, line := range strings.Split(clean, "\n") {
			body.WriteString(`<w:p>` + runText(line, "微软雅黑", "") + `</w:p>`)
		}
	case "params":
		for _, line := range strings.Split(clean, "\n") {
			body.WriteString(`<w:p>` + runText(line, "Consolas", "微软雅黑") + `</w:p>`)
		}
	case "separator":
		body.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="auto"/></w:pBdr></w:pPr></w:p>`)
	case "image":
		// Task 1：占位。Task 2 换成 writeImageSection 真嵌入。
		body.WriteString(imagePlaceholderXML(s.Caption))
	case "link_list":
		for _, it := range s.Items {
			b.linkIdx++
			id := fmt.Sprintf("rIdLink%d", b.linkIdx)
			rels.WriteString(fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>`, id, xmlEscape(it.URL)))
			body.WriteString(`<w:p><w:hyperlink r:id="` + id + `"><w:r><w:rPr><w:rFonts w:ascii="微软雅黑" w:eastAsia="微软雅黑"/><w:color w:val="0000FF"/><w:u w:val="single"/></w:rPr><w:t xml:space="preserve">` + xmlEscape(stripNonBmp(it.Label)) + `</w:t></w:r></w:hyperlink></w:p>`)
		}
	}
}

// runText 生成一个带字体的 run：<w:r><w:rPr>...</w:rPr><w:t>text</w:t></w:r>。
// asciiFont 为西文字体（正文微软雅黑、params Consolas）；eastAsiaFont 为中文字体（空则同 asciiFont）。
func runText(text, asciiFont, eastAsiaFont string) string {
	if eastAsiaFont == "" {
		eastAsiaFont = asciiFont
	}
	return `<w:r><w:rPr><w:rFonts w:ascii="` + asciiFont + `" w:hAnsi="` + asciiFont + `" w:eastAsia="` + eastAsiaFont + `"/></w:rPr><w:t xml:space="preserve">` + xmlEscape(text) + `</w:t></w:r>`
}

// imagePlaceholderXML 生成红字 [图片待补:caption] 段落。caption 空则用"图片"。
func imagePlaceholderXML(caption string) string {
	if caption == "" {
		caption = "图片"
	}
	return `<w:p><w:r><w:rPr><w:rFonts w:ascii="微软雅黑" w:eastAsia="微软雅黑"/><w:color w:val="FF0000"/></w:rPr><w:t xml:space="preserve">[图片待补:` + xmlEscape(caption) + `]</w:t></w:r></w:p>`
}

// stripNonBmp 剥离 BMP 外字符（emoji 如 🚀），保留 BMP 内文字。
func stripNonBmp(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r <= 0xFFFF {
			b.WriteRune(r)
		} else if utf8.RuneLen(r) > 0 {
			// BMP 外（emoji）跳过
		}
	}
	return b.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func writeZip(w *zip.Writer, name, content string) error {
	fw, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = fw.Write([]byte(content))
	return err
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/agent/ -run TestBuildDocx -v`
Expected: PASS（2 个子测试）。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/docx.go backend-go/internal/agent/docx_test.go
git commit -m "feat(agent): add minimal pure-Go docx writer (text sections + link_list)"
```

---

### Task 2: docx 写入器 — image section（嵌入 + 20cm 缩放 + 缺图占位）

把 Task 1 的 image 占位分支换成真嵌入：读 PNG、解码取宽高、按 20cm 高度上限等比缩放、嵌入 `word/media/imageN.png` + 关系 + drawing XML；path 读不到仍走红字占位。引入 `pendingImage` + `writeZipBytes` 收集图片并在主流程写 zip。

**Files:**
- Modify: `backend-go/internal/agent/docx.go`（加 `pendingImage`/`writeZipBytes`/`writeImageSection`，改 `writeSection` 的 image case，`BuildDocx` 收集图片写 media）
- Test: `backend-go/internal/agent/docx_test.go`（加 image 测试 + `png200x100` helper）

**Interfaces:**
- Consumes: `image/png`、`image`、`bytes`、`os`（已 import）+ Task 1 的 `docxBuilder`/`docxSection`/`writeZip`/`xmlEscape`/`imagePlaceholderXML`/`stripNonBmp`。
- Produces: image section 真嵌入；`BuildDocx` 完整。Task 4 的 `buildDocxTool` 调用。

- [ ] **Step 1: 写失败测试**

追加到 `backend-go/internal/agent/docx_test.go`（import 区补 `"bytes"`、`"image"`、`"image/color"`、`"image/png"`）：
```go
func TestBuildDocx_ImageEmbedded(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "card.png")
	if err := os.WriteFile(pngPath, png200x100(), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "with-image.docx")
	sections := []docxSection{
		{Type: "heading", Text: "产品派息与敲出观察点位表"},
		{Type: "image", Path: pngPath, Caption: "产品卡"},
	}
	if err := BuildDocx(sections, out); err != nil {
		t.Fatalf("BuildDocx: %v", err)
	}
	zr, _ := zip.OpenReader(out)
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["word/media/image1.png"] {
		t.Error("图片未嵌入到 word/media/image1.png")
	}
	doc := readZipFile(t, zr, "word/document.xml")
	if !strings.Contains(doc, "产品派息与敲出观察点位表") {
		t.Error("标题缺失")
	}
	if !strings.Contains(doc, "rIdImg1") {
		t.Error("document.xml 缺图片 drawing 引用 rIdImg1")
	}
	rels := readZipFile(t, zr, "word/_rels/document.xml.rels")
	if !strings.Contains(rels, "media/image1.png") {
		t.Error("rels 缺 image1.png 关系")
	}
}

func TestBuildDocx_ImageMissingPlaceholder(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "no-image.docx")
	sections := []docxSection{
		{Type: "image", Path: filepath.Join(dir, "nonexistent.png"), Caption: "一页通"},
	}
	if err := BuildDocx(sections, out); err != nil {
		t.Fatalf("BuildDocx: %v", err)
	}
	zr, _ := zip.OpenReader(out)
	defer zr.Close()
	doc := readZipFile(t, zr, "word/document.xml")
	if !strings.Contains(doc, "[图片待补:一页通]") {
		t.Errorf("缺图应插红字占位，got: %s", doc)
	}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "word/media/") {
			t.Errorf("缺图不应嵌入 media 文件，found %s", f.Name)
		}
	}
}

// png200x100 返回一张 200x100 纯色 PNG 的字节。
func png200x100() []byte {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	png.Encode(&buf, img)
	return buf.Bytes()
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend-go && go test ./internal/agent/ -run TestBuildDocx_Image -v`
Expected: FAIL（Task 1 的 image 分支只写占位，不会嵌入 image1.png；`png200x100` 未定义）。

- [ ] **Step 3: 写实现（image section）**

`backend-go/internal/agent/docx.go`：
1. import 区补 `"bytes"`、`"image"`、`"image/png"`。
2. 在 `docxLink` 之后加 `pendingImage` 类型：
```go
// pendingImage 是待写入 word/media 的图片。BuildDocx 主流程收集后写 zip。
type pendingImage struct {
	Target string
	Data   []byte
}
```
3. 把 `BuildDocx` 的 body/rels 装配段改为收集 images 并在写 document.xml 前先写 media（替换原 `for _, s := range sections { b.writeSection(&body, &rels, s) }` 段为）：
```go
	images := []pendingImage{}
	var body strings.Builder
	var rels strings.Builder
	for _, s := range sections {
		b.writeSection(&body, &rels, s, &images)
	}
	// 写图片 media 文件
	for _, p := range images {
		if err := writeZipBytes(zw, "word/"+p.Target, p.Data); err != nil {
			return err
		}
	}
```
4. 把 `writeSection` 签名改为 `func (b *docxBuilder) writeSection(body, rels *strings.Builder, s docxSection, images *[]pendingImage)`，并把 `case "image":` 分支改为：
```go
	case "image":
		b.imgIdx++
		if err := writeImageSection(body, rels, s, b.imgIdx, images); err != nil {
			body.WriteString(imagePlaceholderXML(s.Caption))
		}
```
5. 在 `imagePlaceholderXML` 之后加 `writeImageSection` 与 `writeZipBytes`：
```go
// writeImageSection 读 PNG、解码取宽高、按 20cm 高度上限等比缩放、写 rels + drawing XML、并把图片数据 append 到 images（由 BuildDocx 写 zip）。
// 返回 error 时调用方写红字占位。
func writeImageSection(body, rels *strings.Builder, s docxSection, idx int, images *[]pendingImage) error {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	pxW, pxH := bounds.Dx(), bounds.Dy()
	if pxW <= 0 || pxH <= 0 {
		return fmt.Errorf("invalid image dimensions %dx%d", pxW, pxH)
	}
	// PNG 无物理 DPI，按 96px/inch 换算；20cm 高度上限 = 7200000 EMU。
	const maxH = 7200000
	emuW := int(float64(pxW) / 96.0 * 914400)
	emuH := int(float64(pxH) / 96.0 * 914400)
	if emuH > maxH {
		scale := float64(maxH) / float64(emuH)
		emuH = maxH
		emuW = int(float64(emuW) * scale)
	}
	id := fmt.Sprintf("rIdImg%d", idx)
	target := fmt.Sprintf("media/image%d.png", idx)
	*images = append(*images, pendingImage{Target: target, Data: data})
	rels.WriteString(fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="%s"/>`, id, target))
	body.WriteString(fmt.Sprintf(`<w:p><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:effectExtent l="0" t="0" r="0" b="0"/><wp:docPr id="%d" name="image%d"/><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="%d" name="image%d"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`, emuW, emuH, idx, idx, idx, idx, id, emuW, emuH))
	if s.Caption != "" {
		body.WriteString(`<w:p><w:r><w:rPr><w:rFonts w:ascii="微软雅黑" w:eastAsia="微软雅黑"/><w:sz w:val="18"/><w:szCs w:val="18"/></w:rPr><w:t xml:space="preserve">` + xmlEscape(stripNonBmp(s.Caption)) + `</w:t></w:r></w:p>`)
	}
	return nil
}

func writeZipBytes(w *zip.Writer, name string, data []byte) error {
	fw, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = fw.Write(data)
	return err
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend-go && go test ./internal/agent/ -run TestBuildDocx -v`
Expected: PASS（4 个子测试：TextSections、StripsNonBmpEmoji、ImageEmbedded、ImageMissingPlaceholder）。

- [ ] **Step 5: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/docx.go backend-go/internal/agent/docx_test.go
git commit -m "feat(agent): docx image embedding (PNG decode, 20cm scale, missing placeholder)"
```

---

### Task 3: `screenshot_amac` + `screenshot_product_card` chromedp 截图

复用 S2 的 `browser.go` chromedp 基建：AMAC 整页截图（JS 异步等值）、通毓产品点位卡（填表→复制为图片→剪贴板读/元素截图兜底）。live = 人工验收。

**Files:**
- Modify: `backend-go/internal/agent/browser.go`（加 `screenshotAMAC` + `screenshotProductCard` + `selectProductType` + `productCardFields` + `readClipboardPNG`）

**Interfaces:**
- Consumes: `newBrowserContext`（S2）、`loginTongyu`/`hasCaptcha`/`fillByLabel`（S2）、`formField`（S2）、`stringArg`、`chromedp`、`os`、`fmt`、`strings`、`time`、`context`、`encoding/base64`。
- Produces: `func screenshotAMAC(url, outPath string) error`、`func screenshotProductCard(params map[string]any, creds tongyuCreds, chromePath, outPath string) error`。Task 4 的 `(s *Service) screenshotAMACTool`/`screenshotProductCardTool` 调用。PNG 落 `public/poster-artifacts/` 下。

- [ ] **Step 1: 加 `screenshotAMAC`**

`backend-go/internal/agent/browser.go` 末尾追加：
```go
// screenshotAMAC 打开 AMAC 详情页（JS 异步加载值），等值出现后整页截图存 outPath。
// url 用 references/amac-manager.md 的模板（type=1 管理人 / type=2 产品）。
func screenshotAMAC(url, outPath string) error {
	ctx, cancel := newBrowserContext(context.Background(), "")
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second), // 等 JS 填值
	); err != nil {
		return err
	}
	var buf []byte
	if err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
		return err
	}
	return os.WriteFile(outPath, buf, 0o644)
}
```

- [ ] **Step 2: 加 `screenshotProductCard` + helpers**

`browser.go` 追加：
```go
// screenshotProductCard 进通毓"产品点位"小工具，按参数填表→提交→复制为图片→取图存 outPath。
// 剪贴板读不到时回退元素截图。流程见 references/product-position-card.md。
func screenshotProductCard(params map[string]any, creds tongyuCreds, chromePath, outPath string) error {
	ctx, cancel := newBrowserContext(context.Background(), chromePath)
	defer cancel()
	if err := loginTongyu(ctx, creds); err != nil {
		return err
	}
	if hasCaptcha(ctx) {
		return fmt.Errorf("遇验证码，产品卡截图失败")
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://terminal.tongyu-quant.com/smallTool/index.html#/product-position"),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return err
	}
	// 选产品类型（原生 select）
	if pt := stringArg(params, "structure_type"); pt != "" {
		_ = chromedp.Run(ctx, chromedp.SetAttribute(`//select`, "value", selectProductType(pt), chromedp.BySearch))
	}
	// 按标签填数值字段
	for _, f := range productCardFields(params) {
		_ = fillByLabel(ctx, f.Label, f.Value)
	}
	// 点提交
	if err := chromedp.Run(ctx, chromedp.Click(`//button/span[contains(text(),'提交')]`, chromedp.BySearch)); err != nil {
		return err
	}
	chromedp.Sleep(1 * time.Second)
	// 点"复制为图片"
	_ = chromedp.Run(ctx, chromedp.Click(`//button/span[contains(text(),'复制为图片')]`, chromedp.BySearch))
	// 优先：剪贴板读 PNG
	if png, err := readClipboardPNG(ctx); err == nil && len(png) > 0 {
		return os.WriteFile(outPath, png, 0o644)
	}
	// 兜底：结果卡元素截图
	var buf []byte
	shotCtx, cancelShot := context.WithTimeout(ctx, 10*time.Second)
	defer cancelShot()
	if err := chromedp.Run(shotCtx, chromedp.Screenshot(`//*[contains(@class,'product-result')]`, &buf, chromedp.BySearch)); err != nil {
		return fmt.Errorf("剪贴板与元素截图均失败: %w", err)
	}
	return os.WriteFile(outPath, buf, 0o644)
}

// selectProductType 把 structure_type 映射成通毓产品点位页 select 的值（DCN/锁盈）。
func selectProductType(structure string) string {
	s := strings.ToLower(structure)
	if strings.Contains(s, "锁盈") || strings.Contains(s, "经典") {
		return "锁盈"
	}
	return "DCN"
}

// productCardFields 把产品参数映射成产品点位页表单字段标签+值（references/product-position-card.md 表）。
func productCardFields(params map[string]any) []formField {
	get := func(k string) string { return stringArg(params, k) }
	var out []formField
	add := func(label, key string) {
		if v := get(key); v != "" {
			out = append(out, formField{Label: label, Value: v})
		}
	}
	add("期限(月)", "期限")
	add("锁定期(月)", "锁定期")
	add("保证金(%)", "保证金")
	add("敲出线(%)", "期初敲出线")
	add("降敲(每月)", "降敲")
	add("降落伞(%)", "降落伞")
	add("每月或有派息(%)", "费后派息")
	add("派息线(%)", "派息线")
	add("入场点位", "current_price")
	return out
}

// readClipboardPNG 用浏览器 evaluate 调 navigator.clipboard.read 取 image/png，返回 PNG 字节。
func readClipboardPNG(ctx context.Context) ([]byte, error) {
	var b64 string
	js := `(async () => {
		const items = await navigator.clipboard.read();
		for (const it of items) {
			for (const type of it.types) {
				if (type === 'image/png') {
					const blob = await it.getType(type);
					const arr = new Uint8Array(await blob.arrayBuffer());
					let s = ''; for (let i = 0; i < arr.length; i++) s += String.fromCharCode(arr[i]);
					return btoa(s);
				}
			}
		}
		return '';
	})()`
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &b64)); err != nil {
		return nil, err
	}
	if b64 == "" {
		return nil, fmt.Errorf("clipboard 无 image/png")
	}
	return base64.StdEncoding.DecodeString(b64)
}
```
import 区补 `"encoding/base64"`（若缺）。

- [ ] **Step 3: 编译验证（不跑 live）**

Run: `cd backend-go && go build ./... && go test ./internal/agent/ -v`
Expected: 编译通过；已有测试仍绿（截图路径不单测）。

- [ ] **Step 4: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/browser.go
git commit -m "feat(agent): add screenshot_amac + screenshot_product_card (chromedp, manual-acceptance)"
```

- [ ] **Step 5: 人工验收（live，用户执行——本环境无法访问 AMAC/通毓）**

前置：`TONGYU_USER`/`TONGYU_PASS` + 能访问 AMAC/通毓的网络。AMAC 截图：调 `screenshot_amac` 传管理人/产品 URL，验证 PNG 生成且含公示字段值。产品卡：调 `screenshot_product_card` 传产品参数，验证 PNG 生成（剪贴板或元素截图）。选择器首次 live 运行按 `references/amac-manager.md`/`product-position-card.md` 调。

---

### Task 4: 注册 3 个工具 + systemPrompt Word 步骤 + e2e

把 `build_docx`/`screenshot_amac`/`screenshot_product_card` 接进 agent；systemPrompt 加 Word 步骤指令。

**Files:**
- Modify: `backend-go/internal/agent/service.go`（executeTool switch + toolDefinitions + systemPrompt + 3 个 `(s *Service)` 方法 + `parseManifest`/`nextPublicID`）

**Interfaces:**
- Consumes: `BuildDocx`（Task 1/2）、`screenshotAMAC`/`screenshotProductCard`（Task 3）、`stringArg`、`fmt`、`os`、`time`、`config.Config`（TongyuUser/Pass/ChromePath）。
- Produces: 3 个 agent 工具；systemPrompt Word 步骤。

- [ ] **Step 1: 加 `(s *Service)` 方法 + helpers**

`service.go` 在 `fetchWinrate` 方法之后追加：
```go
// screenshotAMACTool 是 agent 工具入口：按 AMAC URL 截图，返回 /public URL + path。
func (s *Service) screenshotAMACTool(args map[string]any) map[string]any {
	url := stringArg(args, "url")
	if url == "" {
		return map[string]any{"error": "url is required"}
	}
	id := nextPublicID()
	outPath := fmt.Sprintf("public/poster-artifacts/%s.png", id)
	if err := screenshotAMAC(url, outPath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"url": "/public/poster-artifacts/" + id + ".png", "path": outPath}
}

// screenshotProductCardTool 是 agent 工具入口：按产品参数生成通毓产品点位卡图。
func (s *Service) screenshotProductCardTool(args map[string]any) map[string]any {
	if s.cfg.TongyuUser == "" || s.cfg.TongyuPass == "" {
		return map[string]any{"error": "未配置 TONGYU_USER/TONGYU_PASS，无法取产品卡图"}
	}
	id := nextPublicID()
	outPath := fmt.Sprintf("public/poster-artifacts/%s.png", id)
	if err := screenshotProductCard(args, tongyuCreds{User: s.cfg.TongyuUser, Pass: s.cfg.TongyuPass}, s.cfg.ChromePath, outPath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"url": "/public/poster-artifacts/" + id + ".png", "path": outPath}
}

// buildDocxTool 是 agent 工具入口：按 manifest 装配 .docx，返回 /public 下载 URL。
func (s *Service) buildDocxTool(args map[string]any) map[string]any {
	sections, err := parseManifest(args)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	id := nextPublicID()
	outPath := fmt.Sprintf("public/推介材料/%s.docx", id)
	if err := os.MkdirAll("public/推介材料", 0o755); err != nil {
		return map[string]any{"error": err.Error()}
	}
	if err := BuildDocx(sections, outPath); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"url": "/public/推介材料/" + id + ".docx", "path": outPath}
}

func nextPublicID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// parseManifest 把 agent 传入的 manifest（args["sections"] 数组）解析成 docxSection 切片。
// 每个 section: {type, text, path, caption, items:[{label,url}]}。
func parseManifest(args map[string]any) ([]docxSection, error) {
	raw, ok := args["sections"]
	if !ok {
		return nil, fmt.Errorf("sections is required")
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("sections must be an array")
	}
	var out []docxSection
	for _, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		sec := docxSection{
			Type:    stringArg(m, "type"),
			Text:    stringArg(m, "text"),
			Path:    stringArg(m, "path"),
			Caption: stringArg(m, "caption"),
		}
		if items, ok := m["items"].([]any); ok {
			for _, it := range items {
				im, ok := it.(map[string]any)
				if !ok {
					continue
				}
				sec.Items = append(sec.Items, docxLink{Label: stringArg(im, "label"), URL: stringArg(im, "url")})
			}
		}
		out = append(out, sec)
	}
	return out, nil
}
```
（`service.go` 已 import `fmt`/`time`/`os`；确认。）

- [ ] **Step 2: `executeTool` switch 加 3 case**

`service.go` `executeTool` switch（`fetch_winrate` case 旁）加：
```go
	case "screenshot_amac":
		return s.screenshotAMACTool(args)
	case "screenshot_product_card":
		return s.screenshotProductCardTool(args)
	case "build_docx":
		return s.buildDocxTool(args)
```

- [ ] **Step 3: `toolDefinitions` 加 3 项**

`service.go` `toolDefinitions()`（`fetch_winrate` 定义旁）加：
```go
		{
			Type: "function",
			Function: map[string]any{
				"name":        "screenshot_amac",
				"description": "截 AMAC（amac.org.cn）管理人/产品公示页整页图。URL 形如 https://www.amac.org.cn/index/qzss/details/?type=1&code=<管理人登记编号> 或 type=2&code=<产品编码>&ctype=P。用于推介材料的管理人/产品公示图。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{"type": "string", "description": "AMAC 详情页 URL"},
					},
					"required": []string{"url"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "screenshot_product_card",
				"description": "用通毓终端产品点位小工具按产品参数生成产品结构解析卡图。用于推介材料的派息敲出观察点位表图。需 TONGYU 凭证。structure_type/参数同 fetch_winrate。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"structure_type": map[string]any{"type": "string"},
						"标的":           map[string]any{"type": "string"},
						"期限":           map[string]any{"type": "string"},
						"锁定期":          map[string]any{"type": "string"},
						"期初敲出线":        map[string]any{"type": "string"},
						"降敲":           map[string]any{"type": "string"},
						"降落伞":          map[string]any{"type": "string"},
						"派息线":          map[string]any{"type": "string"},
						"费后派息":         map[string]any{"type": "string"},
						"保证金":          map[string]any{"type": "string"},
						"current_price": map[string]any{"type": "string"},
					},
					"required": []string{"structure_type", "标的", "期限", "期初敲出线", "降落伞", "费后派息", "保证金", "current_price"},
				},
			},
		},
		{
			Type: "function",
			Function: map[string]any{
				"name":        "build_docx",
				"description": "按 manifest 装配 Word 推介材料 .docx。sections 数组每项 {type,text,path,caption,items}。type: heading/subheading/body/params/image/separator/link_list。image 的 path 用 screenshot_amac/screenshot_product_card 返回的 path；缺图自动插红字占位，不报错。用户要出 Word 材料时调本工具。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"sections": map[string]any{"type": "array", "description": "章节数组，按 docx-template.md 顺序：长版/短版文案→公告群通知→派息敲出观察点位表图→胜率数据→一页通→管理人公示图→产品公示图→托管募集账户→销售常见问题"},
					},
					"required": []string{"sections"},
				},
			},
		},
```

- [ ] **Step 4: systemPrompt 加 Word 步骤**

`service.go` `systemPrompt` 常量末尾追加一段（保持单 const 字符串，`\n` 拼接）：
`当用户要出 Word 推介材料/材料时：按 references/docx-template.md 的 10 章顺序组装 manifest 调 build_docx。文案用 fetch_quote/calc_points/fetch_winrate 已算的真实数字，build_docx 只装配不改数字。管理人/产品公示图调 screenshot_amac（传 AMAC URL），产品点位卡调 screenshot_product_card。一页通/托管募集账户/胜率结果截图等用户手动贴或 v1 未取的，留 image 占位（缺图自动红字 [图片待补]，不报错）。`

- [ ] **Step 5: 编译 + 全 agent 测试**

Run: `cd backend-go && go build ./... && go test ./internal/agent/ -v`
Expected: 编译通过；全部测试 PASS（含 Task 1/2 docx 测试 + S1/S2 测试）。

- [ ] **Step 6: 人工验收（docx dry-run，本环境可跑）**

启动后端，触发一次要求出 Word 的对话（用户给完整参数 + 历史底部 + 胜率 98.17%）。Expected：agent 调 fetch_quote→calc_points→fetch_winrate→（screenshot_amac/screenshot_product_card 在本环境会失败返回 error，agent 应跳过或留 image 占位）→build_docx，SSE 出现 `tool_call: build_docx`，返回 `{"url":"/public/推介材料/<id>.docx"}`。下载该 .docx，用 Word 打开：长版/短版文案、参数块、标题、超链接列表正常；缺图处显示红字 [图片待补:...]。live AMAC/产品卡截图见 Task 3 Step 5（用户执行）。

- [ ] **Step 7: 提交**

```bash
cd D:/projects/business-workbench
git add backend-go/internal/agent/service.go
git commit -m "feat(agent): register build_docx/screenshot_amac/screenshot_product_card + Word prompt"
```

---

## Self-Review

**1. Spec coverage:**
- `build_docx` 自写纯 Go docx 写入器 → Task 1/2。✅（spec 说 unioffice 是错的——本计划改为自写，已在 Global Constraints 标注）
- 7 种 section（heading/subheading/body/params/image/separator/link_list）→ Task 1（文本 6 种）+ Task 2（image）。✅
- 图片 20cm 高度上限缩放 + 缺图红字占位 → Task 2。✅
- emoji 剥离 + 字体（雅黑/Consolas）→ Task 1。✅
- `screenshot_amac`（AMAC JS 异步 + 整页截图）→ Task 3。✅
- `screenshot_product_card`（通毓小工具 + 剪贴板读/元素截图兜底）→ Task 3。✅
- 注册 3 工具 + systemPrompt Word 步骤 → Task 4。✅
- manifest 内联（不走 copy_file）→ Task 4 parseManifest + buildDocxTool。✅
- 数字零例外（build_docx 只装配不改数字）→ Task 4 systemPrompt 硬约束 + BuildDocx 无数字逻辑。✅
- 截图 live = 人工验收 → Task 3 Step 5 + Global Constraints。✅
- docx 单测（zip 结构 + XML + 图片嵌入 + 占位）→ Task 1/2 测试。✅

**2. Placeholder scan:** 无 TBD/TODO。docx XML/drawing/rels 全具体；image section 真嵌入 + 缺图占位有测试锁定。Task 3 chromedp 选择器按参考文档语义写 + 明示 live 调参。parseManifest/nextPublicID/writeZipBytes/writeImageSection 全具体。Task 1 image case 写占位、Task 2 替换为真嵌入——衔接明确。

**3. Type consistency:** `docxSection{Type,Text,Path,Caption string; Items []docxLink}` 在 Task 1 定义、Task 2/4 用。`docxLink{Label,URL string}` 在 Task 1 定义、Task 4 parseManifest 构造。`docxBuilder{imgIdx,linkIdx int}` 在 Task 1 定义、Task 2 用 imgIdx。`BuildDocx(sections []docxSection, outputPath string) error` 在 Task 1 定义、Task 1 测试 + Task 4 buildDocxTool 调用。`pendingImage{Target,Data}` 在 Task 2 定义、Task 2 用。`writeZip`（Task 1）/`writeZipBytes`（Task 2）签名明确。`writeSection(body, rels, s, images *[]pendingImage)`（Task 2 改后签名）在 Task 2 定义、BuildDocx 调用。`writeImageSection(body, rels, s, idx int, images *[]pendingImage) error` 在 Task 2 定义、writeSection image case 调用。`formField`（S2 定义）在 Task 3 productCardFields 复用。`tongyuCreds`（S2 定义）在 Task 3 screenshotProductCard + Task 4 screenshotProductCardTool 复用。`screenshotAMAC(url,outPath) error` / `screenshotProductCard(params,creds,chromePath,outPath) error` 在 Task 3 定义、Task 4 方法调用。工具名 `build_docx`/`screenshot_amac`/`screenshot_product_card` 在 Task 4 switch 与 toolDefinitions 一致。`(s *Service)` 方法名 `screenshotAMACTool`/`screenshotProductCardTool`/`buildDocxTool` 在 Task 4 定义与 switch 调用一致（带 Tool 后缀避免与 chromedp 函数 `screenshotAMAC`/`screenshotProductCard` 同名冲突）。
