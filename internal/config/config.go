package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address       string
	DatabaseURL   string
	RedisAddress  string
	RedisPassword string
	RedisDB       int
	MasterKey     string
	CookieSecure  bool
	SessionTTL    time.Duration
}

func Load() (Config, error) {
	db, err := strconv.Atoi(env("REDIS_DB", "0"))
	if err != nil { return Config{}, fmt.Errorf("REDIS_DB: %w", err) }
	ttl, err := time.ParseDuration(env("SESSION_TTL", "24h"))
	if err != nil { return Config{}, fmt.Errorf("SESSION_TTL: %w", err) }
	cfg := Config{
		Address: env("APP_ADDRESS", "127.0.0.1:8090"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisAddress: env("REDIS_ADDRESS", "127.0.0.1:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB: db,
		MasterKey: os.Getenv("MASTER_KEY"),
		CookieSecure: env("COOKIE_SECURE", "true") == "true",
		SessionTTL: ttl,
	}
	if cfg.DatabaseURL == "" { return Config{}, fmt.Errorf("DATABASE_URL is required") }
	if len(cfg.MasterKey) < 32 { return Config{}, fmt.Errorf("MASTER_KEY must contain at least 32 characters") }
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" { return value }
	return fallback
}
