package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"example.com/go-chat/hub"
	"example.com/go-chat/logger"
	"example.com/go-chat/pkg/auth"
	"example.com/go-chat/pkg/chatter"
	"example.com/go-chat/pkg/template"
	"example.com/go-chat/user"
	"example.com/go-chat/ws"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	logger logger.Logger
}

func NewServer(logger logger.Logger) *Server {

	return &Server{
		logger: logger,
	}
}

func (s *Server) Start() {

	serverCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	r := chi.NewRouter()

	db, err := sql.Open("sqlite3", "chat.db")
	if err != nil {
		s.logger.Errorf("open db: %v", err)
		os.Exit(1)
	}

	// initialize message hub
	hub := hub.NewHub(serverCtx, s.logger)
	wsClientFactory := ws.NewWSClientFactory(hub, s.logger)

	// initialize the template store
	storePath, err := filepath.Abs("templates")
	if err != nil {
		s.logger.Errorf("get abs path: %v", err)
		os.Exit(1)
	}
	templStore := template.NewTemplStore(storePath)

	userStore := user.NewSQLiteUserStore(db)

	secret := make([]byte, base64.StdEncoding.DecodedLen(len("lfxZPXMooyBXiaiQjdU1QLtHHFQ09Z0zSjhvTxVW5XQ=")))
	base64.StdEncoding.Decode(secret, []byte("lfxZPXMooyBXiaiQjdU1QLtHHFQ09Z0zSjhvTxVW5XQ="))
	_auth := auth.NewSimpleAuth(userStore, db, auth.TokenOptions{
		Secret: secret,
		Exp:    time.Hour,
	})

	authHandler := auth.NewAuthHandler(_auth, templStore, s.logger)

	userHandler := user.NewUserHandle(userStore, templStore, s.logger)

	r.Get("/ws", wsClientFactory.HandleFunc)

	r.Group(func(r chi.Router) {
		r.Use(authHandler.JWTAuthMiddleware)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			session, ok := auth.SessionFromContext(r.Context())
			if !ok {
				chatter.HTMXRedirect(w, r, "/users/signin", http.StatusTemporaryRedirect)
				return
			}

			if err := templStore.Render(w, "index", struct {
				Session auth.Session
			}{
				Session: session,
			}); err != nil {
				s.logger.Errorf("render template: %v", err)
			}
		})

		r.Get("/chats/new", func(w http.ResponseWriter, r *http.Request) {
			_, ok := auth.SessionFromContext(r.Context())
			if !ok {
				chatter.HTMXRedirect(w, r, "/users/signin", http.StatusTemporaryRedirect)
				return
			}

			if err := templStore.Render(w, "chats/new", nil); err != nil {
				s.logger.Errorf("render template: %v", err)
			}
		})

		r.Get("/users", userHandler.GetUsers)

	})

	r.Get("/users/signin", func(w http.ResponseWriter, r *http.Request) {
		if err := templStore.Render(w, "users/signin", nil); err != nil {
			s.logger.Errorf("render template: %v", err)
		}
	})

	r.Post("/users/signin", authHandler.SigninHandler)

	r.Get("/users/signup", func(w http.ResponseWriter, r *http.Request) {
		if err := templStore.Render(w, "users/signup", nil); err != nil {
			s.logger.Errorf("render template: %v", err)

		}
	})

	r.Post("/users/signup", userHandler.CreateUserHandler)

	r.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		if err := templStore.Render(w, "error/index", nil); err != nil {
			s.logger.Errorf("render template: %v", err)
		}
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
		BaseContext: func(_ net.Listener) context.Context {
			return serverCtx

		},
	}

	done := make(chan struct{})

	go func() {
		<-serverCtx.Done()

		s.logger.Infof("Server is shutting down\n")

		exitCtx, _ := context.WithTimeout(serverCtx, 20*time.Second)

		go func() {
			<-exitCtx.Done()
			if exitCtx.Err() == context.DeadlineExceeded {
				s.logger.Errorf("gracefull shotdown timed out.. forcing exit.")
				os.Exit(1)
			}

		}()

		err := server.Shutdown(exitCtx)
		if err != nil {
			s.logger.Errorf("server shutdown: %v", err)
			os.Exit(1)
		}

		db.Close()
		close(done)
	}()

	s.logger.Infof("server started at %s", server.Addr)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		s.logger.Errorf("server exit: %v", err)
		os.Exit(1)

	}
	<-done
}
