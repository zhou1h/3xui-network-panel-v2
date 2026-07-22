package main

import (
	"log"
	"time"

	"github.com/hibiken/asynq"

	appcore "github.com/zhou1h/3xui-network-panel-v2/internal/app"
	"github.com/zhou1h/3xui-network-panel-v2/internal/config"
	"github.com/zhou1h/3xui-network-panel-v2/internal/tasks"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	application, err := appcore.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: cfg.RedisAddress, Password: cfg.RedisPassword, DB: cfg.RedisDB}, asynq.Config{Concurrency: 2, ShutdownTimeout: 15 * time.Second})
	mux := asynq.NewServeMux()
	(&tasks.Handler{App: application}).Register(mux)
	if err := server.Run(mux); err != nil {
		log.Fatal(err)
	}
}
