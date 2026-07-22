package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/hibiken/asynq"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"

	"github.com/zhou1h/3xui-network-panel-v2/ent"
	"github.com/zhou1h/3xui-network-panel-v2/internal/config"
	"github.com/zhou1h/3xui-network-panel-v2/internal/security"
)

type App struct {
	Config config.Config
	DB     *ent.Client
	SQL    *sql.DB
	Redis  *redis.Client
	Queue  *asynq.Client
	Cipher *security.Cipher
}

func Open(cfg config.Config) (*App, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(driver))
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddress, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	queue := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddress, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	cipherBox, err := security.NewCipher(cfg.MasterKey)
	if err != nil {
		return nil, err
	}
	return &App{Config: cfg, DB: client, SQL: db, Redis: rdb, Queue: queue, Cipher: cipherBox}, nil
}

func (a *App) Ready(ctx context.Context) error {
	if err := a.SQL.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if err := a.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	return nil
}

func (a *App) Close() { _ = a.Queue.Close(); _ = a.Redis.Close(); _ = a.DB.Close() }
