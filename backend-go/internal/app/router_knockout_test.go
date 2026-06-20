package app

import "testing"

func TestParseFirstKnockoutRatio_AbsoluteETFPriceToRatio(t *testing.T) {
	got := parseFirstKnockoutRatio("0.77353", 0.751)
	want := 1.03
	if got < want-0.00001 || got > want+0.00001 {
		t.Fatalf("got %.6f want %.6f", got, want)
	}
}

func TestParseFirstKnockoutRatio_RatioStringKeepsRatio(t *testing.T) {
	got := parseFirstKnockoutRatio("1.03", 0.751)
	if got != 1.03 {
		t.Fatalf("got %.6f want 1.03", got)
	}
}

func TestParseFirstKnockoutRatio_PercentStringToRatio(t *testing.T) {
	got := parseFirstKnockoutRatio("103%", 0.751)
	if got != 1.03 {
		t.Fatalf("got %.6f want 1.03", got)
	}
}

func TestParseDaifanTaxRatios_ParsesOutstandingFromHeaders(t *testing.T) {
	raw := [][]any{
		{"订单号", "管理费实收", "业绩报酬应收", "管理费扣税", "业绩报酬扣税", "未返管理费", "未返业绩报酬"},
		{"snowball-A", "200", "300", "0.1", "0.2", "50", "70"},
		{"申购费返还"},
		{"订单号", "申购费扣税", "未返申购费"},
		{"snowball-A", "0.05", "80"},
	}

	got := parseDaifanTaxRatios(raw)
	if got.mgmtReceived["snowball-A"] != 200 {
		t.Fatalf("mgmt received = %v, want 200", got.mgmtReceived["snowball-A"])
	}
	if got.perfReceivable["snowball-A"] != 300 {
		t.Fatalf("perf receivable = %v, want 300", got.perfReceivable["snowball-A"])
	}
	if got.subTax["snowball-A"] != 0.05 {
		t.Fatalf("sub tax = %v, want 0.05", got.subTax["snowball-A"])
	}
	if got.mgmtOutstanding["snowball-A"] != 50 {
		t.Fatalf("mgmt outstanding = %v, want 50", got.mgmtOutstanding["snowball-A"])
	}
	if got.perfOutstanding["snowball-A"] != 70 {
		t.Fatalf("perf outstanding = %v, want 70", got.perfOutstanding["snowball-A"])
	}
	if got.subOutstanding["snowball-A"] != 80 {
		t.Fatalf("sub outstanding = %v, want 80", got.subOutstanding["snowball-A"])
	}
}

func TestParseDaifanTaxRatios_FallsBackToConfirmedOutstandingColumns(t *testing.T) {
	raw := [][]any{
		{"订单号", "航班编号", "航班名称", "客户姓名", "返还人", "本金", "管理费实收", "业绩报酬应收", "管理费", "业绩报酬", "管理费", "业绩报酬", "管理费", "业绩报酬", "管理费", "业绩报酬", "管理费", "业绩报酬", "是否可返"},
		{"snowball-B", "081", "X", "Y", "Z", "1000000", "10000", "2000", "0.25", "0.25", "0.15", "0.15", "2125", "425", "0", "", "2125", "425", "是"},
		{"申购费返还"},
		{"订单号", "航班编号", "航班名称", "客户姓名", "返还人", "本金", "已收申购费", "返还比例", "扣税比例", "应返申购费", "已返", "未返", "是否可返"},
		{"snowball-B", "081", "X", "Y", "Z", "1000000", "5000", "1", "0.15", "4250", "0", "4250", "是"},
	}

	got := parseDaifanTaxRatios(raw)
	if got.mgmtOutstanding["snowball-B"] != 2125 {
		t.Fatalf("mgmt outstanding = %v, want 2125", got.mgmtOutstanding["snowball-B"])
	}
	if got.perfOutstanding["snowball-B"] != 425 {
		t.Fatalf("perf outstanding = %v, want 425", got.perfOutstanding["snowball-B"])
	}
	if got.subOutstanding["snowball-B"] != 4250 {
		t.Fatalf("sub outstanding = %v, want 4250", got.subOutstanding["snowball-B"])
	}
}
