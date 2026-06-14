package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	FrontendURL       string
	DatabasePath      string
	FeishuAppID       string
	FeishuAppSecret   string
	FeishuRedirectURI string
	DeepSeekAPIKey    string
	DeepSeekAPIURL    string
	DeepSeekModel     string
	CronTimezone      string
	FeishuPushWebhook string
	SMTPHost          string
	SMTPPort          string
	SMTPSecure        string
	SMTPUser          string
	SMTPPass          string
	SMTPFrom          string
}

func Load() Config {
	_ = godotenv.Load(".env")

	smtpUser := os.Getenv("SMTP_USER")
	return Config{
		Port:              getEnv("PORT", "3001"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:5173"),
		DatabasePath:      getEnv("DATABASE_PATH", "data.sqlite"),
		FeishuAppID:       os.Getenv("FEISHU_APP_ID"),
		FeishuAppSecret:   os.Getenv("FEISHU_APP_SECRET"),
		FeishuRedirectURI: os.Getenv("FEISHU_REDIRECT_URI"),
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

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
