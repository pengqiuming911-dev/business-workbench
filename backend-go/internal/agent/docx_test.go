package agent

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
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
