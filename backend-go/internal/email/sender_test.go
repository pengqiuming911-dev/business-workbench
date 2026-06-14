package email

import (
	"strings"
	"testing"

	"business-workbench/backend-go/internal/model"
)

func TestRenderText(t *testing.T) {
	products := []NotificationProduct{
		{
			Product:       model.Product{ID: "F001", Name: "测试产品", Manager: "管理人A", Code: "000001"},
			EntryPriceStr: "100.00",
			Obs:           ObservationSnapshot{UnderlyingPriceStr: "105.50", KnockoutPriceStr: "110.00", DividendLineStr: "95.00", IsKnockedOut: "否", IsDividend: "是"},
		},
	}
	text := renderText("2026-06-13", products)
	for _, want := range []string{"2026-06-13", "测试产品", "管理人A", "否", "是"} {
		if !strings.Contains(text, want) {
			t.Errorf("renderText missing %q\n%s", want, text)
		}
	}
}

func TestRenderHtml(t *testing.T) {
	products := []NotificationProduct{
		{
			Product:       model.Product{ID: "F001", Name: "测试产品", Manager: "管理人A", Code: "000001"},
			EntryPriceStr: "100.00",
			Obs:           ObservationSnapshot{UnderlyingPriceStr: "105.50", KnockoutPriceStr: "110.00", DividendLineStr: "95.00", IsKnockedOut: "否", IsDividend: "是"},
		},
	}
	html := renderHtml("2026-06-13", products)
	if !strings.Contains(html, "<table") || !strings.Contains(html, "</table>") {
		t.Error("renderHtml should produce a table")
	}
	if !strings.Contains(html, "测试产品") {
		t.Error("renderHtml should include product name")
	}
}

func TestEscapeHtml(t *testing.T) {
	input := `<script>alert("xss")</script>`
	got := escapeHtml(input)
	if strings.Contains(got, "<") || strings.Contains(got, ">") || strings.Contains(got, `"`) {
		t.Errorf("escapeHtml should sanitize, got %q", got)
	}
	if !strings.Contains(got, "&lt;") {
		t.Error("escapeHtml should replace < with &lt;")
	}
}

func TestBuildTodayNotification_NoProducts(t *testing.T) {
	cfg := Config{SMTPHost: "", SMTPUser: "", SMTPPass: ""}
	n := BuildTodayNotification(nil, nil, "2026-06-13", cfg)
	if n.Subject == "" {
		t.Error("subject should not be empty")
	}
	if len(n.Products) != 0 {
		t.Errorf("expected 0 products, got %d", len(n.Products))
	}
}

func TestSendObservationEmail_NoSMTP(t *testing.T) {
	cfg := Config{}
	n := &Notification{Products: []NotificationProduct{{}}}
	sent, reason := SendObservationEmail(cfg, n)
	if sent {
		t.Error("should not send when SMTP not configured")
	}
	if reason != "smtp-not-configured" {
		t.Errorf("expected smtp-not-configured, got %s", reason)
	}
}

func TestSendObservationEmail_NoProducts(t *testing.T) {
	cfg := Config{SMTPHost: "smtp.example.com", SMTPUser: "u", SMTPPass: "p"}
	n := &Notification{Products: nil}
	sent, reason := SendObservationEmail(cfg, n)
	if sent {
		t.Error("should not send when no products")
	}
	if reason != "no-product" {
		t.Errorf("expected no-product, got %s", reason)
	}
}
