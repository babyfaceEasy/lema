package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/db"

	//_ "github.com/babyfaceeasy/lema/docs"
	"github.com/babyfaceeasy/lema/internal/server"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/babyfaceeasy/lema/internal/tasks"
	"go.uber.org/zap"
)

// @title Github Service
// @version 1.0
// @description This is a simple service that fetches data fom Github's API.
// @host localhost:3000
// @BasePath /
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	defer logger.Sync()

	// load configurations
	conf, err := config.New()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// db
	conn, err := db.NewPostgresDb(conf)
	if err != nil {
		return err
	}

	dataStore := store.New(conn)

	// run migrations
	err = db.RunMigrations(conn, logger)
	if err != nil {
		return err
	}

	// start worker / task server
	tsk := tasks.New(conf, logger, dataStore)
	go func() {
		if err := tasks.StartWorker(*tsk, conf); err != nil {
			// return err
			logger.Fatal("worker server encountered and error", zap.Error(err))
		}
	}()

	// create and start server
	svr := server.New(conf, logger, dataStore)
	if err := svr.Start(ctx); err != nil {
		return err
	}

	return nil
}
