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
	config      *Config
	db          *core.SQLiteDB
	context     context.Context
	server      *http.Server
	logger      *slog.Logger
	router      *router.Router
	eventRouter *core.EventRouter
	wsManager   *core.ConnManager

	exit chan int

	userStore core.UserStore
	chatStore core.ChatStore
	authStore core.AuthStore

	userHandler *UserHandler
	chatHandler *ChatHandler
	authhandler *AuthHandler

	cleanupFuncs []func(context.Context)

	staticFS *StaticFS

	wg sync.WaitGroup
}

func New(ctx context.Context, config *Config, staticFS *StaticFS) *App {
	var err error
	app := &App{
		exit: make(chan int),
	}
	if ctx == nil {
		ctx, _ = signal.NotifyContext(
			context.Background(),
			syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	}
	app.context = ctx

	if config == nil {
		var err error
		config, err = LoadConfig()
		if err != nil {
			failed(1, "failed to load config: %v\n", err)
		}
	}
	if err := config.Validate(); err != nil {
		failed(1, FormatValidationErrors(err))
	}
	app.config = config

	app.staticFS = staticFS

	app.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug,
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
	app.db, err = core.NewSQLiteDB(app.config.SQLite.File, app.config.SQLite.Migrations, sqliteOptions)
	if err != nil {
		failed(1, "failed to open database: %v\n", err)
	}
	app.AddCleanupFunc(func(ctx context.Context) {
		app.db.Close()
	})
	if err := app.db.Migrate(); err != nil {
		failed(1, "failed to migrate database: %v\n", err)
	}

	app.userStore = core.NewSqlieUserStore(app.db.DB)
	app.authStore = core.NewSQLiteAuthStore(app.db.DB, app.userStore, []byte(app.config.Auth.Secret))
	app.chatStore = core.NewSQLiteChatStore(app.db.DB, app.userStore)

	app.wsManager = core.NewConnManager(app.context, &app.wg, app.logger)
	app.wsManager.OnUserConnected(app.onUserConnect)
	app.wsManager.OnConnectionOpened(app.onConnectionOpen)
	app.wsManager.OnUserDisconnected(app.onUserDisconnect)
	app.eventRouter = core.NewEventRouter(app.context, app.logger, app.wsManager)
	app.eventRouter.On(MessageEvent, app.MessageEventHandler)
	app.eventRouter.On(ReadMessageEvent, app.ReadMessageHandler)
	app.eventRouter.On(TypingEvent, app.TypingHandler)
	app.eventRouter.On(IsOnlineEvent, app.IsOnlineHandler)

	app.userHandler = NewUserHandler(app.userStore)
	app.chatHandler = NewChatHandler(app.chatStore)
	app.authhandler = NewAuthHandler(app.authStore)
	authMiddleware := JWTMiddleware(app.authStore)

	app.router = router.New(router.WithLogger(app.logger))

	app.router.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.config.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	app.router.With(authMiddleware).Router.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		session := core.SessionFromRequest(r)
		err := app.wsManager.Connect(session.Username, w, r)
		if err != nil {
			return
		}
	})

	api := router.New(router.WithLogger(app.logger))

	api.Route("/users", func(r *router.Router) {
		r.With(authMiddleware).Get("/me", app.userHandler.MeHandler)
		r.Post("/", app.userHandler.RegisterUserHandler)
		r.Get("/{username}", app.userHandler.GetUserByUsernameHandler)
	})

	api.Group(func(r *router.Router) {
		r.Use(authMiddleware)
		r.Get("/users/me/rooms", app.chatHandler.GetMyRoomsHandler)
		r.Get("/rooms/{roomID}", app.chatHandler.GetRoomByIDHandler)
		r.Post("/rooms", app.chatHandler.CreateRoomHandler)
		r.Get("/rooms/{roomID}/messages", app.chatHandler.GetRoomMessagesHandler)
		r.Post("/rooms/{roomID}/members", app.chatHandler.AddRoomMemberHandler)
		r.Delete("/rooms/{roomID}/members/{userID}", app.chatHandler.RemoveRoomMemberHandler)
	})

	api.Route("/auth", func(r *router.Router) {
		r.Post("/signin", app.authhandler.SigninHandler)
		r.Post("/signout", app.authhandler.SignoutHandler)
	})

	app.router.Mount("/api", api)

	if app.staticFS != nil {
		app.router.Router.With(staticFS.EtagMiddleware()).Mount("/", http.FileServer(staticFS))
	}

	app.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", app.config.Hostname, app.config.Port),
		Handler: app.router.Router,
		BaseContext: func(listener net.Listener) context.Context {
			return app.context
		},
	}
	if app.config.Mode == ProdMode {
		app.server.TLSConfig = &defaultTLSConfig
	}

	return app
}

func (app *App) Start() {
	app.eventRouter.Listen()
	app.AddCleanupFunc(func(ctx context.Context) {
		app.eventRouter.Close(ctx)
	})

	// listen for shutdown signal
	go func() {
		<-app.context.Done()
		close(app.exit)
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		var wg sync.WaitGroup

		for _, f := range app.cleanupFuncs {
			wg.Add(1)
			func(wg *sync.WaitGroup) {
				defer wg.Done()
				f(closeCtx)
			}(&wg)

		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			app.logger.Info("app shutdown gracefully")
			app.exit <- 0
		case <-closeCtx.Done():
			app.logger.Info("app shutdown timed out")
			app.exit <- 1

		}

	}()

	app.AddCleanupFunc(func(ctx context.Context) {
		app.server.Shutdown(ctx)
	})
	app.logger.Info(fmt.Sprintf("app running in %s mode on: %s:%d",
		app.config.Mode, app.config.Hostname, app.config.Port))

	var err error
	// TODO: perhaps better validation for TLS config
	if app.config.TLS.Key != "" && app.config.TLS.Crt != "" {
		err = app.server.ListenAndServeTLS(app.config.TLS.Crt, app.config.TLS.Key)
	} else {

		err = app.server.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		failed(1, "server error: %v\n", err)
	}

	code := <-app.exit
	if code != 0 {
		failed(code, "app exit with code: %d\n", code)
	} else {
		os.Exit(code)
	}

}

func (app *App) AddCleanupFunc(f func(context.Context)) {
	app.cleanupFuncs = append(app.cleanupFuncs, f)
}

func failed(code int, s string, args ...interface{}) {
	fmt.Printf(s, args...)
	os.Exit(code)
}
