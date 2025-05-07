package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"medods-test/internal/api"
	"medods-test/internal/config"
	"medods-test/internal/logger"
	"medods-test/internal/storage/postgres"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "medods-test/docs"

	"github.com/golang-migrate/migrate/v3"
	_ "github.com/golang-migrate/migrate/v3/database/postgres"
	_ "github.com/golang-migrate/migrate/v3/source/file"
)

const (
	InfoDbClosed = "Storage is closed. App is shuting down"
)

// @title medods-test
// @version 1.0
// @description Auth service

// @host localhost:8080
// @BasePath /api/v1/
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

	// err = startMigrations(log, cfg.DbConnString)
	// if err != nil {
	// 	log.Error("can't start migrations", "err", err)

	// 	os.Exit(1)
	// }

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

func startMigrations(log *slog.Logger, connString string) error {
	m, err := migrate.New("file://migrations", connString) // DEBUG: ../../migrations"
	if err != nil {
		return fmt.Errorf("can't start migration driver:%w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Warn("Migarate didn't run. Nothing to change")
			return nil
		}
		return fmt.Errorf("failed to do migrations:%w", err)

	}

	return nil
}
