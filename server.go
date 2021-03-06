package anansi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
)

func listenForInterrupt(log zerolog.Logger, cancel context.CancelFunc) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// wait for Quit signal
	<-quit

	// cancel context so dependencies can gracefully shutdown
	log.Info().Msg("Signal caught. Shutting down...")
	cancel()
}

func waitForContext(ctx context.Context, log zerolog.Logger, server *http.Server) chan struct{} {
	// listen for shutdown signal from the context so we can shutdown the server
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		if err := server.Shutdown(context.Background()); err != nil {
			log.Err(err).Msg("Could not shut down server cleanly.....")
		}
		close(done)
	}()

	return done
}

func RunServer(port int, log zerolog.Logger, handler http.Handler, close func() error) {
	// request context
	ctx, cancel := context.WithCancel(context.Background())
	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     handler,
		ReadTimeout: 2 * time.Minute,
	}

	go listenForInterrupt(log, cancel)
	done := waitForContext(ctx, log, server)

	log.Info().Msgf("Serving api at http://127.0.0.1:%d", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Err(err).Msg("Could not start the server")
	}

	// return only when done is closed
	<-done

	// close application resource, only logging errors
	if err := close(); err != nil {
		log.Warn().Err(err).Msg("Could not close app resources")
	}
}
