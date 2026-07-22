package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/hibiken/asynq"

	appcore "github.com/zhou1h/3xui-network-panel-v2/internal/app"
	"github.com/zhou1h/3xui-network-panel-v2/internal/config"
)

func main() {
	cfg, err := config.Load(); if err != nil { log.Fatal(err) }
	application, err := appcore.Open(cfg); if err != nil { log.Fatal(err) }; defer application.Close()
	server := asynq.NewServer(asynq.RedisClientOpt{Addr:cfg.RedisAddress, Password:cfg.RedisPassword, DB:cfg.RedisDB}, asynq.Config{Concurrency:2, ShutdownTimeout:15*time.Second})
	mux := asynq.NewServeMux()
	mux.HandleFunc("resource:health", func(ctx context.Context, task *asynq.Task) error { var payload map[string]any; if err:=json.Unmarshal(task.Payload(),&payload); err!=nil{return err}; log.Printf("resource health task: %v",payload); return nil })
	if err := server.Run(mux); err != nil { log.Fatal(err) }
	_ = application
}
