package api

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/go-chat/pkg/logger"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	logger logger.Logger
}

func NewServer() *Server {

	logger := logger.NewDefault()
	return &Server{
		logger: logger,
	}
}

func (s *Server) Start() {

	serverCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	db, err := sql.Open("sqlite3", "chat.db")
	if err != nil {
		s.logger.Errorf("open db: %v", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr: ":8080",
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
