package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhou1h/3xui-network-panel-v2/ent/user"
	appcore "github.com/zhou1h/3xui-network-panel-v2/internal/app"
	"github.com/zhou1h/3xui-network-panel-v2/internal/config"
	"github.com/zhou1h/3xui-network-panel-v2/internal/httpapi"
	"github.com/zhou1h/3xui-network-panel-v2/internal/security"
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := application.DB.Schema.Create(ctx); err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "bootstrap-admin" {
		bootstrapAdmin(ctx, application)
		return
	}
	if err := application.Ready(ctx); err != nil {
		log.Fatal(err)
	}
	router := httpapi.New(application)
	serverErr := make(chan error, 1)
	go func() { serverErr <- router.Run(cfg.Address) }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		log.Fatal(err)
	case <-signals:
		log.Println("shutdown requested")
	}
}

func bootstrapAdmin(ctx context.Context, application *appcore.App) {
	username, password := os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("ADMIN_USERNAME and ADMIN_PASSWORD are required")
	}
	exists, err := application.DB.User.Query().Where(user.UsernameEQ(username)).Exist(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		log.Fatal("administrator already exists")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}
	account, err := application.DB.User.Create().SetUsername(username).SetPasswordHash(hash).SetRole("owner").SetMustChangePassword(false).Save(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("administrator created: %s (id=%d)\n", account.Username, account.ID)
}
