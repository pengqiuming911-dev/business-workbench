package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	FrontendURL         string
	DatabasePath        string
	FeishuAppID         string
	FeishuAppSecret     string
	FeishuRedirectURI   string
	AllowedFeishuEmails []string
	DeepSeekAPIKey      string
	DeepSeekAPIURL      string
	DeepSeekModel       string
	CronTimezone        string
	FeishuPushWebhook   string
	SMTPHost            string
	SMTPPort            string
	SMTPSecure          string
	SMTPUser            string
	SMTPPass            string
	SMTPFrom            string
}

func Load() Config {
	_ = godotenv.Load(".env", filepath.Join("backend-go", ".env"))

	smtpUser := os.Getenv("SMTP_USER")
	return Config{
		Port:              getEnv("PORT", "3001"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:5173"),
		DatabasePath:      resolveDatabasePath(),
		FeishuAppID:       os.Getenv("FEISHU_APP_ID"),
		FeishuAppSecret:   os.Getenv("FEISHU_APP_SECRET"),
		FeishuRedirectURI: os.Getenv("FEISHU_REDIRECT_URI"),
		AllowedFeishuEmails: parseListEnv("FEISHU_ALLOWED_EMAILS", []string{
			"lvjunliang@iyanxuan.cn",
			"fanweifeng@iyanxuan.cn",
			"zhaochunhui@iyanxuan.cn",
			"pengqiuming@iyanxuan.cn",
		}),
		DeepSeekAPIKey:    os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekAPIURL:    getEnv("DEEPSEEK_API_URL", "https://api.deepseek.com"),
		DeepSeekModel:     getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		CronTimezone:      getEnv("CRON_TIMEZONE", "Asia/Shanghai"),
		FeishuPushWebhook: os.Getenv("FEISHU_PUSH_WEBHOOK"),
		SMTPHost:          os.Getenv("SMTP_HOST"),
		SMTPPort:          getEnv("SMTP_PORT", "587"),
		SMTPSecure:        getEnv("SMTP_SECURE", "false"),
		SMTPUser:          smtpUser,
		SMTPPass:          os.Getenv("SMTP_PASS"),
		SMTPFrom:          getEnv("SMTP_FROM", smtpUser),
	}
}

func resolveDatabasePath() string {
	if value := os.Getenv("DATABASE_PATH"); value != "" {
		return value
	}
	candidates := []string{
		"data.sqlite",
		filepath.Join("backend-go", "data.sqlite"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "data.sqlite"
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseListEnv(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '\t':
			return true
		default:
			return false
		}
	})
	var out []string
	seen := map[string]struct{}{}
	for _, part := range parts {
		value := strings.ToLower(strings.TrimSpace(part))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
