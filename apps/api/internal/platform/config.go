package platform

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct{ DatabaseURL, RedisURL, JWTSecret, SiteURL, Origin, Address, Env, MailDriver, SMTPHost, SMTPPort, SMTPUser, SMTPPassword, SMTPFrom, UploadDir, RevalidationURL, RevalidationSecret string }

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func Load() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), RedisURL: env("REDIS_URL", "redis://localhost:6379/0"), JWTSecret: os.Getenv("JWT_SECRET"), SiteURL: env("SITE_URL", "http://localhost:3000"), Origin: env("CORS_ORIGIN", "http://localhost:3000"), Address: env("HTTP_ADDR", ":8080"), Env: env("APP_ENV", "development"), MailDriver: env("MAIL_DRIVER", "mock"), SMTPHost: os.Getenv("SMTP_HOST"), SMTPPort: env("SMTP_PORT", "587"), SMTPUser: os.Getenv("SMTP_USERNAME"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: os.Getenv("SMTP_FROM")}
	if c.DatabaseURL == "" || len(c.JWTSecret) < 32 {
		return c, fmt.Errorf("DATABASE_URL and JWT_SECRET (at least 32 bytes) required")
	}
	c.UploadDir = env("UPLOAD_DIR", "./uploads")
	c.RevalidationURL = os.Getenv("WEB_REVALIDATION_URL")
	c.RevalidationSecret = os.Getenv("REVALIDATION_SECRET")
	if c.RevalidationURL != "" {
		u, e := url.Parse(c.RevalidationURL)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || len(c.RevalidationSecret) < 32 {
			return c, fmt.Errorf("WEB_REVALIDATION_URL and REVALIDATION_SECRET (at least 32 bytes) required")
		}
	}
	for _, v := range []string{c.SiteURL, c.Origin} {
		u, e := url.Parse(v)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return c, fmt.Errorf("SITE_URL/CORS_ORIGIN must be an origin without trailing slash")
		}
	}
	if c.MailDriver != "mock" && c.MailDriver != "smtp" {
		return c, fmt.Errorf("unsupported MAIL_DRIVER")
	}
	if c.MailDriver == "smtp" && (c.SMTPHost == "" || c.SMTPFrom == "") {
		return c, fmt.Errorf("SMTP_HOST and SMTP_FROM required")
	}
	if c.Env == "production" && (!strings.HasPrefix(c.SiteURL, "https://") || !strings.HasPrefix(c.Origin, "https://") || c.MailDriver != "smtp") {
		return c, fmt.Errorf("production requires HTTPS origins and SMTP")
	}
	return c, nil
}
