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
