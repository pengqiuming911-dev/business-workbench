package agent

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image/png"
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

// pendingImage 是待写入 word/media 的图片。BuildDocx 主流程收集后写 zip。
type pendingImage struct {
	Target string
	Data   []byte
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

func (b *docxBuilder) writeSection(body, rels *strings.Builder, s docxSection, images *[]pendingImage) {
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
		// Task 2：真嵌入（path 读不到仍走红字占位）。
		b.imgIdx++
		if err := writeImageSection(body, rels, s, b.imgIdx, images); err != nil {
			body.WriteString(imagePlaceholderXML(s.Caption))
		}
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
	return `<w:p><w:r><w:rPr><w:rFonts w:ascii="微软雅黑" w:eastAsia="微软雅黑"/><w:color w:val="FF0000"/></w:rPr><w:t xml:space="preserve">[图片待补:` + xmlEscape(stripNonBmp(caption)) + `]</w:t></w:r></w:p>`
}

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
