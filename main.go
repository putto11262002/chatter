package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-chi/cors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/putto11262002/chatter/handlers"
	"github.com/putto11262002/chatter/pkg/router"
	"github.com/putto11262002/chatter/pkg/server"
	"github.com/putto11262002/chatter/pkg/ws"
	"github.com/putto11262002/chatter/store"
	"github.com/putto11262002/chatter/ws"
)

// port
// auth secret
// token expiration
// db file

type Config struct {
	// Port is the Port number to listen on. The default is 8080.
	Port int
	// Hostname is the Hostname to listen on. The default is 0.0.0.0.
	Hostname string
	// Secret is the Secret key used to sign JWT tokens.
	// This must be a base64 encoded string.
	Secret []byte
	// SQLiteFile is the path to the SQLite database file.
	SQLiteFile string
	// AllowedOrigins is a list of origins that are allowed to connect to the server.
	AllowedOrigins []string
}

func loadConfig() (*Config, error) {
	config := &Config{
		Port:     8080,
		Hostname: "0.0.0.0",
	}

	portStr := os.Getenv("PORT")
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse port: %v", err)
		}
		if port > 0 && port < 65535 {
			config.Port = port
		}
	}

	hostname := os.Getenv("HOSTNAME")
	if hostname != "" {
		config.Hostname = hostname
	}

	secretStr := os.Getenv("SECRET")
	if secretStr == "" {
		return nil, fmt.Errorf("SECRET is required")
	}
	secret, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(secretStr)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret: %v", err)
	}
	config.Secret = secret

	dbFile := os.Getenv("SQLITE_FILE")
	if dbFile == "" {
		return nil, fmt.Errorf("SQLITE_FILE is required")
	}
	config.SQLiteFile = dbFile

	originsStr := os.Getenv("ALLOWED_ORIGINS")
	origins := strings.Split(originsStr, ",")
	config.AllowedOrigins = origins

	return config, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config, err := loadConfig()
	if err != nil {
		logger.Error(fmt.Sprintf("failed to load config: %v", err))
		os.Exit(1)
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared", config.SQLiteFile))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to open database: %v", err))
		os.Exit(1)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		logger.Error(fmt.Sprintf("failed to migrate database: %v", err))
		os.Exit(1)
	}

	userStore := store.NewSqlieUserStore(db)
	authStore := store.NewSQLiteAuthStore(db, userStore, []byte(config.Secret))
	chatStore := store.NewSQLiteChatStore(db, userStore)

	authHandler := handlers.NewAuthHandler(authStore)
	userHandler := handlers.NewUserHandler(userStore)
	chatHandler := handlers.NewChatHandler(chatStore)

	hub := hub.New(
		hub.WithLogger(logger),
		hub.WithBaseContext(ctx),
		hub.WithAuthenticator(&ws.Authenticator{}),
	)

	chatWSHandler := ws.NewChatWSHandler(chatStore)
	hub.SetHandle(ws.Message, chatWSHandler.MessageHandler)
	hub.SetHandle(ws.ReadMessage, chatWSHandler.ReadMessage)
	hub.SetHandle(ws.Typing, chatWSHandler.TypingHandler)

	hub.Start()

	r := router.New()
	r.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.With(handlers.JWTMiddleware(authStore)).
		Router.Get("/ws", hub.ServeHTTP)

	api := router.New(router.WithLogger(logger))

	api.Post("/signin", authHandler.SigninHandler)
	api.With(handlers.JWTMiddleware(authStore)).Post("/signout", authHandler.SignoutHandler)
	api.Post("/signup", userHandler.CreateUserHandler)

	api.Route("/users", func(r *router.Router) {
		r.With(handlers.JWTMiddleware(authStore)).Get("/me", userHandler.MeHandler)
		r.With(handlers.JWTMiddleware(authStore)).Get("/{username}", userHandler.GetUserByIDHandler)
	})

	api.Group(func(r *router.Router) {
		r.Use(handlers.JWTMiddleware(authStore))
		r.Get("/users/me/rooms", chatHandler.GetMyRoomSummaries)
		r.Get("/rooms/{roomID}", chatHandler.GetRoomByIDHandler)
		r.Post("/rooms/private", chatHandler.CreatePrivateChatHandler)
		r.Post("/rooms/group", chatHandler.CreateGroupChatHandler)
		r.Get("/rooms/{roomID}/messages", chatHandler.GetRoomMessagesHandler)
		r.Post("/rooms/messages", chatHandler.SendMessageHandler)
	})

	r.Mount("/api", api)

	server := server.New(
		fmt.Sprintf("%s:%d", config.Hostname, config.Port),
		server.WithLogger(logger),
		server.WithHandler(r),
		server.WithBaseContext(ctx),
	)
	server.RegisterOnShutdown(func() {
		hub.Close()
	})
	server.Start(ctx)

	hub.Wait()
}

func migrate(db *sql.DB) error {

	migrationFS := os.DirFS("./migrations")
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(db, "."); err != nil {
		return err
	}
	return nil
}
