package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Server struct {
	*http.Server
	// CleanUpFuncs is a list of functions that will be called when the server has successfully shutdown.
	CleanUpFuncs []func(ctx context.Context)
}

func (s *Server) Start(ctx context.Context) {

	s.Server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}

	done := make(chan struct{})

	go func() {
		<-ctx.Done()

		log.Println("server shuting down...")

		shutdownCtx, _ := context.WithTimeout(context.Background(), 20*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Println("gracefull shotdown timed out.. forcing exit.")
				os.Exit(1)
			}

		}()

		err := s.Server.Shutdown(shutdownCtx)

		// TODO: maybe run the cleanup functions even if there is an error?
		if err != nil {
			log.Println("server shutdown: %v", err)
			os.Exit(1)
		}

		// TODO: run them concurrently?
		for _, cf := range s.CleanUpFuncs {
			cf(shutdownCtx)
		}

		close(done)
	}()

	log.Printf("server started at %s\n", s.Server.Addr)

	err := s.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Println("server exit: %v", err)
		os.Exit(1)

	}

	<-done
}
