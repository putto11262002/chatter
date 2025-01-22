package chatter

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/cors"
	"github.com/putto11262002/chatter/core"
	"github.com/putto11262002/chatter/pkg/router"
	"github.com/putto11262002/chatter/ws"
)

type App struct {
	// configLoader specifies how to load the configuration.
	// The default is DefaultConfigLoader.
	configLoader  ConfigLoader
	config        *Config
	db            *core.SQLiteDB
	context       context.Context
	contextCancel context.CancelFunc
	server        *http.Server
	logger        *slog.Logger
	router        *router.Router
	hub           ws.Hub
	done          chan int

	userStore core.UserStore
	chatStore core.ChatStore
	authStore core.AuthStore

	userHandler *UserHandler
	chatHandler *ChatHandler
	authhandler *AuthHandler
}

func New(config *Config) *App {

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	app := &App{
		configLoader:  &EnvConfigLoader{},
		done:          make(chan int),
		context:       ctx,
		contextCancel: cancel,
		config:        config,
	}
	return app
}

func (a *App) Start() {
	config, err := a.configLoader.Load()
	if err != nil {
		exit(1, "failed to load config: %v\n", err)
	}

	config = MergeConfig(a.config, config)
	err = config.Validate()
	if err != nil {
		exit(1, err.Error())
	}

	a.config = config

	a.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	sqliteOptions := &core.SQLiteDBOption{
		Mode:        "rwc",
		Cache:       "shared",
		JournalMode: "WAL",
	}

	p, err := filepath.Abs(a.config.SQLiteFile)
	if err != nil {
		a.logger.Error(err.Error())
	}
	a.logger.Info(p)

	if a.db, err = core.NewSQLiteDB(a.config.SQLiteFile, a.config.MigrationDir, sqliteOptions); err != nil {
		exit(1, "failed to open database: %v\n", err)
	}

	if err := a.db.Migrate(); err != nil {
		exit(1, "failed to migrate database: %v\n", err)
	}

	a.userStore = core.NewSqlieUserStore(a.db.DB)
	a.authStore = core.NewSQLiteAuthStore(a.db.DB, a.userStore, a.config.Secret)
	a.chatStore = core.NewSQLiteChatStore(a.db.DB, a.userStore)

	authenticator := NewWSAuthenticator(a.authStore)
	a.hub = ws.New(ws.NewWSConnFactory(), authenticator,
		ws.WithLogger(a.logger), ws.WithBaseContext(a.context))

	a.hub.Start()

	a.userHandler = NewUserHandler(a.userStore)
	a.chatHandler = NewChatHandler(a.chatStore)
	a.authhandler = NewAuthHandler(a.authStore)
	authMiddleware := JWTMiddleware(a.authStore)

	a.router = router.New(router.WithLogger(a.logger))
	a.router.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   a.config.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	a.router.With(authMiddleware).Router.Get("/ws", a.hub.ServeHTTP)

	api := router.New(router.WithLogger(a.logger))

	api.Route("/users", func(r *router.Router) {
		r.With(authMiddleware).Get("/me", a.userHandler.MeHandler)
		r.Post("/", a.userHandler.RegisterUserHandler)
		r.Get("/{username}", a.userHandler.GetUserByUsernameHandler)
	})

	api.Group(func(r *router.Router) {
		r.Use(authMiddleware)
		r.Get("/users/me/rooms", a.chatHandler.GetMyRoomsHandler)
		r.Get("/rooms/{roomID}", a.chatHandler.GetRoomByIDHandler)
		r.Post("/rooms", a.chatHandler.CreateRoomHandler)
		r.Get("/rooms/{roomID}/messages", a.chatHandler.GetRoomMessagesHandler)
		r.Post("/rooms/{roomID}/members", a.chatHandler.AddRoomMemberHandler)
		r.Delete("/rooms/{roomID}/members/{userID}", a.chatHandler.RemoveRoomMemberHandler)
	})

	api.Route("/auth", func(r *router.Router) {
		r.Post("/signin", a.authhandler.SigninHandler)
		r.Post("/signout", a.authhandler.SignoutHandler)
	})

	a.router.Mount("/api", api)

	// listen for shutdown signal
	go func() {
		<-a.context.Done()
		a.close()
	}()

	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.Hostname, a.config.Port),
		Handler: a.router.Router,
		BaseContext: func(listener net.Listener) context.Context {
			return a.context
		},
	}

	a.logger.Info(fmt.Sprintf("app running on: %s:%d", a.config.Hostname, a.config.Port))
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		exit(1, "server error: %v\n", err)
	}

	code := <-a.done
	exit(code, "")
}

func (a *App) Stop() {
	a.contextCancel()
}

func (a *App) close() {
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer closeCancel()
	var wg sync.WaitGroup
	// close hub
	wg.Add(1)
	go func() {
		a.hub.Close()
		wg.Done()
	}()

	// close server
	wg.Add(1)
	go func() {
		a.server.Shutdown(closeCtx)
		wg.Done()
	}()

	// close db
	wg.Add(1)
	go func() {
		a.db.Close()
		wg.Done()
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("app shutdown gracefully")
		a.done <- 0
	case <-closeCtx.Done():
		a.logger.Info("app shutdown timed out")
		a.done <- 1

	}
}

func exit(code int, s string, args ...interface{}) {
	fmt.Printf(s, args...)
	os.Exit(code)
}
