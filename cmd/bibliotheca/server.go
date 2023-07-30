package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/emzola/bibliotheca/internal/jsonlog"
)

func (a *app) serve(wg *sync.WaitGroup, logger *jsonlog.Logger) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.Server.Port),
		Handler:      a.handler.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	shutdownError := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})
		wg.Wait()
		shutdownError <- nil
	}()

	// Start server and listen for incoming connections
	logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  a.config.Server.Env,
	})
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	err = <-shutdownError
	if err != nil {
		return err
	}
	logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})
	return nil
}
