// Package config читает конфигурацию из окружения с безопасными дефолтами.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr     string
	StaffListenAddr string
	DatabaseURL    string
	AttachmentsDir string
	CookieSecure   bool
	SessionTTL     time.Duration
	SeedDefaultPwd string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	secure := false
	if v := os.Getenv("COOKIE_SECURE"); v == "1" || v == "true" {
		secure = true
	}
	ttl := 2 * time.Hour
	if v := os.Getenv("SESSION_TTL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Minute
		}
	}
	return Config{
		ListenAddr:      env("LISTEN_ADDR", ":8080"),
		StaffListenAddr: env("STAFF_LISTEN_ADDR", ":8081"),
		DatabaseURL:     env("DATABASE_URL", "postgres://otklik:otklik@localhost:5432/otklik?sslmode=disable"),
		AttachmentsDir:  env("ATTACHMENTS_DIR", "./data/attachments"),
		CookieSecure:    secure,
		SessionTTL:      ttl,
		SeedDefaultPwd:  env("SEED_DEFAULT_PWD", "otklik-demo-2026"),
	}
}
