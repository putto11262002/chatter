package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/go-chat/internal/api"
	"example.com/go-chat/pkg/auth"
	"example.com/go-chat/pkg/server"
	"github.com/go-chi/chi/v5"
	"github.com/pressly/goose/v3"
)

func main() {

	serverCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	db, err := sql.Open("sqlite3", "file:./chat.db")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	migrationFS := os.DirFS("./migrations")
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("dialect: %v", err)
	}

	if err := goose.Up(db, "."); err != nil {
		log.Fatalf("migrate up: %v", err)
	}

	apiConfig := api.ApiConfig{
		TokenOptions: auth.TokenOptions{
			Secret: []byte("secret"),
			Exp:    time.Hour,
		},
	}
	_api := api.NewApi(serverCtx, db, apiConfig)

	r := chi.NewRouter()

	r.Mount("/api", _api.Mux())

	server := server.Server{
		Server: &http.Server{
			Handler: r,
			Addr:    ":8080",
			BaseContext: func(_ net.Listener) context.Context {
				return serverCtx
			},
		},
	}

	server.Start(serverCtx)
}
