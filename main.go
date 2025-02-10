package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"os/signal"
	"syscall"

	chatter "github.com/putto11262002/chatter/app"
)

//go:embed web/dist/*
var static embed.FS

func main() {
	config, err := chatter.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	sub, err := fs.Sub(static, "web/dist")
	if err != nil {
		log.Fatalf("sub static: %v", err)
	}
	staticFS, err := chatter.NewStaticFS(sub, "index.html", map[string]string{
		"index.html":   "no-cache",
		"assets/*.js":  "public, max-age=31536000",
		"assets/*.css": "public, max-age=31536000",
	})
	if err != nil {
		log.Fatalf("create static fs: %v", err)
	}

	context, _ := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	app := chatter.New(context, config, staticFS)

	app.Start()
}
