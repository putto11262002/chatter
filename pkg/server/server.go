package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

// Server is a wrapper around http.Server that implements graceful shutdown.
type Server struct {
	*http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

type ServerOption = func(*Server)

func New(addr string, opts ...ServerOption) *Server {
	server := &Server{
		Server: &http.Server{
			Addr: addr,
		},
	}

	for _, opt := range opts {
		opt(server)
	}

	return server
}

func WithBaseContext(baseCtx context.Context) ServerOption {
	return func(s *Server) {
		s.BaseContext = func(_ net.Listener) context.Context {
			return baseCtx
		}
	}
}

func WithHandler(handler http.Handler) ServerOption {
	return func(s *Server) {
		s.Handler = handler
	}
}

func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

func WithShutdownTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}

func (s *Server) Start(ctx context.Context) {

	s.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	done := make(chan struct{})

	go func() {
		<-ctx.Done()

		s.logger.Info("server shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				s.logger.Info("gracefull shotdown timed out.. forcing exit.")
				os.Exit(1)
			}

		}()

		err := s.Server.Shutdown(shutdownCtx)
		if err != nil {
			s.logger.Error(fmt.Sprintf("server shutdown: %v", err))
			os.Exit(1)
		}

		close(done)
	}()

	s.logger.Info(fmt.Sprintf("server started at %s", s.Server.Addr))
	err := s.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		s.logger.Error(fmt.Sprintf("server exit: %v", err))
		os.Exit(1)

	}

	<-done
}
