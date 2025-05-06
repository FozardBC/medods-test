package main

import (
	"context"
	"medods-test/internal/api"
	"medods-test/internal/config"
	"medods-test/internal/logger"
	"medods-test/internal/storage/postgres"
	"net/http"
	"os"
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

	chanError := make(chan error, 1)

	runServer(serverAddr, api.Router, chanError)
	log.Info("Server is started", "addres", serverAddr)

}

func runServer(addr string, handler http.Handler, errChan chan<- error) {
	srv := http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		errChan <- srv.ListenAndServe()
	}()
}
