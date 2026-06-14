package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"business-workbench/backend-go/internal/model"
	"business-workbench/backend-go/internal/observations"
)

type Config struct {
	SMTPHost   string
	SMTPPort   string
	SMTPSecure string
	SMTPUser   string
	SMTPPass   string
	SMTPFrom   string
}

type ObservationSnapshot struct {
	UnderlyingPriceStr string
	KnockoutPriceStr   string
	DividendLineStr    string
	IsKnockedOut       string
	IsDividend         string
}

type NotificationProduct struct {
	Product       model.Product
	EntryPriceStr string
	Obs           ObservationSnapshot
}

type Notification struct {
	Recipient string
	Subject   string
	Text      string
	HTML      string
	Products  []NotificationProduct
}

const defaultRecipient = "pengqiuming@iyanxuna.cn"

func formatPrice(value *float64) string {
	if value == nil {
		return "--"
	}
	return fmt.Sprintf("%.2f", *value)
}

func formatValue(v interface{}) string {
	if v == nil {
		return "--"
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return "--"
	}
	return s
}

func BuildTodayNotification(products []model.Product, prices map[string]float64, today string, cfg Config) *Notification {
	result := []NotificationProduct{}
	for _, product := range products {
		if product.Code == "" {
			continue
		}
		price, ok := prices[product.Code]
		if !ok {
			continue
		}
		info := todayObsInfo(product, today)
		if info.monthsSinceEntry == 0 {
			continue
		}
		eval := observations.EvaluateObservation(product, today, price, info.monthsSinceEntry)
		result = append(result, NotificationProduct{
			Product:       product,
			EntryPriceStr: formatPrice(product.EntryPrice),
			Obs: ObservationSnapshot{
				UnderlyingPriceStr: fmt.Sprintf("%.2f", eval.UnderlyingPrice),
				KnockoutPriceStr:   formatPrice(eval.KnockoutPrice),
				DividendLineStr:    formatPrice(eval.DividendLine),
				IsKnockedOut:       eval.IsKnockedOut,
				IsDividend:         eval.IsDividend,
			},
		})
	}
	subject := fmt.Sprintf("今日产品派息/敲出观察提醒（%s，%d个）", today, len(result))
	return &Notification{
		Recipient: defaultRecipient,
		Subject:   subject,
		Text:      renderText(today, result),
		HTML:      renderHtml(today, result),
		Products:  result,
	}
}

type todayObs struct {
	date             string
	monthsSinceEntry int
}

func todayObsInfo(product model.Product, today string) todayObs {
	for _, d := range observations.DatesUntil(product, today) {
		if d.Date == today {
			return todayObs{date: d.Date, monthsSinceEntry: d.MonthsSinceEntry}
		}
	}
	return todayObs{}
}

func SendObservationEmail(cfg Config, n *Notification) (sent bool, reason string) {
	if n == nil || len(n.Products) == 0 {
		return false, "no-product"
	}
	if cfg.SMTPHost == "" || cfg.SMTPUser == "" || cfg.SMTPPass == "" {
		return false, "smtp-not-configured"
	}
	from := cfg.SMTPFrom
	if from == "" {
		from = cfg.SMTPUser
	}
	port := cfg.SMTPPort
	if port == "" {
		port = "587"
	}
	addr := cfg.SMTPHost + ":" + port
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	body := buildMIMEBody(from, n.Recipient, n.Subject, n.Text, n.HTML)
	err := smtp.SendMail(addr, auth, from, []string{n.Recipient}, []byte(body))
	if err != nil {
		return false, "send-error: " + err.Error()
	}
	return true, ""
}

func buildMIMEBody(from, to, subject, text, html string) string {
	boundary := "GoMailBoundary"
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n")
	b.WriteString("\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n" + text + "\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n" + html + "\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return b.String()
}

func renderText(today string, products []NotificationProduct) string {
	lines := []string{
		"今日产品派息/敲出观察提醒",
		"观察日期：" + today,
		fmt.Sprintf("今日需要观察产品数量：%d", len(products)),
		"",
	}
	for _, p := range products {
		lines = append(lines,
			"产品："+formatValue(p.Product.Name),
			"航班编号："+formatValue(p.Product.ID),
			"私募管理人："+formatValue(p.Product.Manager),
			"标的代码："+formatValue(p.Product.Code),
			"入场价："+p.EntryPriceStr,
			"实时标的价格："+p.Obs.UnderlyingPriceStr,
			"敲出价："+p.Obs.KnockoutPriceStr,
			"派息线："+p.Obs.DividendLineStr,
			"是否敲出："+p.Obs.IsKnockedOut,
			"是否派息："+p.Obs.IsDividend,
			"",
		)
	}
	return strings.Join(lines, "\n")
}

func renderHtml(today string, products []NotificationProduct) string {
	var rows strings.Builder
	for _, p := range products {
		rows.WriteString(fmt.Sprintf(`<tr>
			<td>%s</td><td>%s</td><td>%s</td><td>%s</td>
			<td style="text-align:right">%s</td><td style="text-align:right">%s</td>
			<td style="text-align:right">%s</td><td style="text-align:right">%s</td>
			<td style="text-align:center">%s</td><td style="text-align:center">%s</td>
		</tr>`,
			escapeHtml(p.Product.ID), escapeHtml(p.Product.Name),
			escapeHtml(p.Product.Manager), escapeHtml(p.Product.Code),
			escapeHtml(p.EntryPriceStr), escapeHtml(p.Obs.UnderlyingPriceStr),
			escapeHtml(p.Obs.KnockoutPriceStr), escapeHtml(p.Obs.DividendLineStr),
			escapeHtml(p.Obs.IsKnockedOut), escapeHtml(p.Obs.IsDividend),
		))
	}
	return fmt.Sprintf(`<div>
		<h2>今日产品派息/敲出观察提醒</h2>
		<p>观察日期：%s；今日需要观察产品数量：%d</p>
		<table border="1" cellpadding="8" cellspacing="0" style="border-collapse:collapse;font-size:13px;">
			<thead><tr>
				<th>航班编号</th><th>产品名称</th><th>私募管理人</th><th>标的代码</th>
				<th>入场价</th><th>实时标的价格</th><th>敲出价</th><th>派息线</th>
				<th>是否敲出</th><th>是否派息</th>
			</tr></thead>
			<tbody>%s</tbody>
		</table>
	</div>`, escapeHtml(today), len(products), rows.String())
}

func escapeHtml(value any) string {
	s := fmt.Sprint(value)
	if value == nil {
		s = "--"
	}
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
