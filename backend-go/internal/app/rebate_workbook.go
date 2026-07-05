package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const rebateWorkbookContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type rebateAccountInfo struct {
	AccountName string
	CardNumber  string
	BankName    string
}

type rebateWorkbookResult struct {
	Data        []byte
	FileName    string
	ContentType string
}

func (s *Server) buildRebateWorkbook(ctx context.Context, items []rebateFlowItem) (rebateWorkbookResult, error) {
	accounts, err := s.loadRebateAccounts(ctx)
	if err != nil {
		return rebateWorkbookResult{}, err
	}
	detailRows := [][]string{{
		"返还人", "订单号", "航班编号", "航班名称", "客户姓名",
		"本次拟返-申购费", "本次拟返-管理费", "本次拟返-业绩报酬", "本次拟返合计",
	}}
	type summary struct {
		RebateTarget string
		Sub          float64
		Mgmt         float64
		Perf         float64
		Total        float64
		Account      rebateAccountInfo
	}
	summaryMap := map[string]*summary{}
	for _, item := range items {
		target := strings.TrimSpace(item.RebateTarget)
		if target == "" {
			target = "未填写返还人"
		}
		sub := positive(item.OutstandingSubscribe)
		mgmt := positive(item.OutstandingManagement)
		perf := positive(item.OutstandingPerformance)
		total := sub + mgmt + perf
		detailRows = append(detailRows, []string{
			target,
			item.OrderID,
			item.FlightID,
			item.ProductName,
			item.CustomerName,
			moneyString(sub),
			moneyString(mgmt),
			moneyString(perf),
			moneyString(total),
		})
		row := summaryMap[target]
		if row == nil {
			row = &summary{RebateTarget: target, Account: accounts[target]}
			summaryMap[target] = row
		}
		row.Sub += sub
		row.Mgmt += mgmt
		row.Perf += perf
		row.Total += total
	}
	summaryRows := [][]string{{
		"返还人", "本次拟返-申购费", "本次拟返-管理费", "本次拟返-业绩报酬", "本次拟返合计",
		"收款账户", "卡号", "银行名称",
	}}
	keys := make([]string, 0, len(summaryMap))
	for key := range summaryMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		row := summaryMap[key]
		summaryRows = append(summaryRows, []string{
			row.RebateTarget,
			moneyString(row.Sub),
			moneyString(row.Mgmt),
			moneyString(row.Perf),
			moneyString(row.Total),
			row.Account.AccountName,
			row.Account.CardNumber,
			row.Account.BankName,
		})
	}
	data, err := xlsxBytes([]xlsxSheet{
		{Name: "拟返明细", Rows: detailRows},
		{Name: "返还人汇总", Rows: summaryRows},
	})
	if err != nil {
		return rebateWorkbookResult{}, err
	}
	return rebateWorkbookResult{
		Data:        data,
		FileName:    fmt.Sprintf("待返费明细_%s.xlsx", strings.ReplaceAll(timeNowDate(), "-", "")),
		ContentType: rebateWorkbookContentType,
	}, nil
}

func (s *Server) loadRebateAccounts(ctx context.Context) (map[string]rebateAccountInfo, error) {
	token := strings.TrimSpace(s.cfg.RebateAccountSheetToken)
	if token == "" {
		return map[string]rebateAccountInfo{}, nil
	}
	metas, err := s.feishu.GetSheetMetaData(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("读取返还人收款账户表失败: %w", err)
	}
	if len(metas) == 0 {
		return map[string]rebateAccountInfo{}, nil
	}
	sheetID := metas[0].SheetID
	colCount := metas[0].GridProps.ColumnCount
	if colCount <= 0 || colCount > 30 {
		colCount = 12
	}
	rows, err := s.feishu.ReadSheetRows(ctx, token, sheetID, colCount)
	if err != nil {
		return nil, fmt.Errorf("读取返还人收款账户表失败: %w", err)
	}
	out := map[string]rebateAccountInfo{}
	for _, row := range rows {
		name := firstRowValue(row, "返还人", "返还对象", "姓名", "客户姓名", "名称")
		if name == "" {
			continue
		}
		out[name] = rebateAccountInfo{
			AccountName: firstRowValue(row, "收款账户", "账户", "户名", "账户名称", "收款人"),
			CardNumber:  firstRowValue(row, "卡号", "账号", "银行卡号", "银行账号", "收款账号"),
			BankName:    firstRowValue(row, "银行名称", "银行", "开户行", "开户银行", "支行"),
		}
	}
	return out, nil
}

func firstRowValue(row map[string]any, names ...string) string {
	for _, name := range names {
		for key, value := range row {
			if normalizeHeader(key) == normalizeHeader(name) {
				text := strings.TrimSpace(fmt.Sprint(value))
				if text != "" && text != "<nil>" {
					return text
				}
			}
		}
	}
	return ""
}

func normalizeHeader(value string) string {
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\t", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "：", "")
	value = strings.ReplaceAll(value, ":", "")
	return strings.ToLower(strings.TrimSpace(value))
}

func positive(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func moneyString(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func timeNowDate() string {
	return time.Now().Format("2006-01-02")
}

type xlsxSheet struct {
	Name string
	Rows [][]string
}

func xlsxBytes(sheets []xlsxSheet) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml":        contentTypesXML(len(sheets)),
		"_rels/.rels":                rootRelsXML(),
		"xl/workbook.xml":            workbookXML(sheets),
		"xl/_rels/workbook.xml.rels": workbookRelsXML(len(sheets)),
		"xl/styles.xml":              stylesXML(),
		"docProps/core.xml":          coreXML(),
		"docProps/app.xml":           appXML(),
	}
	for i, sheet := range sheets {
		files[fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)] = worksheetXML(sheet.Rows)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		w, err := zw.Create(filepath.ToSlash(name))
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func worksheetXML(rows [][]string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		rowNum := r + 1
		b.WriteString(`<row r="` + strconv.Itoa(rowNum) + `">`)
		for c, value := range row {
			ref := colName(c+1) + strconv.Itoa(rowNum)
			b.WriteString(`<c r="` + ref + `" t="inlineStr"><is><t>`)
			b.WriteString(xmlEscape(value))
			b.WriteString(`</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func colName(index int) string {
	var out []byte
	for index > 0 {
		index--
		out = append([]byte{byte('A' + index%26)}, out...)
		index /= 26
	}
	return string(out)
}

func xmlEscape(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func contentTypesXML(sheetCount int) string {
	var overrides strings.Builder
	for i := 1; i <= sheetCount; i++ {
		overrides.WriteString(fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i))
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>` + overrides.String() + `</Types>`
}

func rootRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
}

func workbookXML(sheets []xlsxSheet) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for i, sheet := range sheets {
		b.WriteString(fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, html.EscapeString(sheet.Name), i+1, i+1))
	}
	b.WriteString(`</sheets></workbook>`)
	return b.String()
}

func workbookRelsXML(sheetCount int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= sheetCount; i++ {
		b.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i))
	}
	b.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, sheetCount+1))
	b.WriteString(`</Relationships>`)
	return b.String()
}

func stylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts><fills count="1"><fill><patternFill patternType="none"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="1"><xf xfId="0"/></cellXfs></styleSheet>`
}

func coreXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>待返费明细</dc:title></cp:coreProperties>`
}

func appXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>Business Workbench</Application></Properties>`
}
