package main

import (
	"context"
	"medods-test/internal/api"
	"medods-test/internal/config"
	"medods-test/internal/logger"
	"medods-test/internal/storage/postgres"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	InfoDbClosed = "Storage is closed. App is shuting down"
)

func main() {

	ctx := context.Background()

	cfg := config.MustRead()

	serverAddr := cfg.ServerHost + ":" + cfg.ServerPort

	log := logger.New(cfg.Log)

	storage, err := postgres.New(ctx, log, cfg.DbConnString)
	if err != nil {
		log.Error("can't connect to storage", "err", err.Error())

		os.Exit(1)
	}

	api := api.New(log, storage)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	chanError := make(chan error, 1)

	srv := http.Server{
		Addr:    serverAddr,
		Handler: api.Router,
	}

	go func() {
		chanError <- srv.ListenAndServe()
	}()
	log.Info("Server is started", "addres", serverAddr)

	select {
	case err := <-chanError:
		log.Error("Shutting down. Critical error:", "err", err)

		shutdown <- syscall.SIGTERM
	case sig := <-shutdown:
		log.Error("received signal, starting graceful shutdown", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("server graceful shutdown failed", "err", err)
			err = srv.Close()
			if err != nil {
				log.Error("forced shutdown failed", "err", err)
			}
		}

		storage.Close()

		log.Info(InfoDbClosed)

		log.Info("shutdown completed")

	}

}
