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
	eventRouter   *core.EventRouter
	wsManager     *core.ConnManager

	done chan int

	userStore core.UserStore
	chatStore core.ChatStore
	authStore core.AuthStore

	userHandler *UserHandler
	chatHandler *ChatHandler
	authhandler *AuthHandler

	wg sync.WaitGroup
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

	a.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source, _ := a.Value.Any().(*slog.Source)
				if source != nil {
					source.File = filepath.Base(source.File)
				}
			}
			return a
		},
	}))

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

	a.wsManager = core.NewConnManager(a.context, &a.wg, a.logger)
	// TODO:
	a.wsManager.OnUserConnected(func(username string) {
		// if user is not already connected, send a message to all friends that user is online
		friends, err := a.chatStore.GetFriends(a.context, username)
		if err != nil {
			return
		}
		payload := OnlineEventPayload{Username: username}
		a.eventRouter.EmitTo(OnlineEvent, payload, friends...)

	})

	a.wsManager.OnConnectionOpened(func(username string, i int) {

		friends, err := a.chatStore.GetFriends(a.context, username)
		if err != nil {
			return
		}
		// now send the online status of all friends to the user
		for _, friend := range friends {
			connected := a.wsManager.IsUserConnected(friend)
			if connected {
				payload := OnlineEventPayload{Username: friend}
				a.eventRouter.EmitTo(OnlineEvent, payload, username)
			}
		}
	})

	a.wsManager.OnUserDisconnected(func(username string) {
		friends, err := a.chatStore.GetFriends(a.context, username)
		if err != nil {
			return
		}
		payload := OfflineEventPayload{Username: username}
		a.eventRouter.EmitTo(OfflineEvent, payload, friends...)
	})

	a.eventRouter = core.NewEventRouter(a.context, a.logger, a.wsManager)
	a.wg.Add(1)
	go a.eventRouter.Listen(&a.wg)
	a.eventRouter.On(MessageEvent, a.MessageEventHandler)
	a.eventRouter.On(ReadMessageEvent, a.ReadMessageHandler)
	a.eventRouter.On(TypingEvent, a.TypingHandler)

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

	a.router.With(authMiddleware).Router.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		session := core.SessionFromRequest(r)
		err := a.wsManager.Connect(session.Username, w, r)
		if err != nil {
			return
		}
	})

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
