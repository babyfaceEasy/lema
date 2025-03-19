package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/db"
	"github.com/babyfaceeasy/lema/pkg/logger"

	"github.com/babyfaceeasy/lema/internal/container"
	"github.com/babyfaceeasy/lema/internal/server"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/babyfaceeasy/lema/internal/tasks"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// load configurations
	// cfg, err := config.New()
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// logger
	logr, err := logger.NewLogger(string(cfg.GetAppEnv()))
	if err != nil {
		return err
	}
	defer logr.Sync()

	// db
	// todo: delete later
	conn, err := db.NewPostgresDb(cfg)
	if err != nil {
		return err
	}

	dataStore := store.New(conn)

	// container
	diContainer := container.NewContainer(cfg, logr)
	defer diContainer.Close()

	// run migrations
	err = db.RunMigrations(conn, logr)
	if err != nil {
		return err
	}

	// start worker / task server
	tsk := tasks.New(cfg, logr, dataStore, diContainer.GetCommitService(), diContainer.GetRepositoryService())
	go func() {
		if err := tasks.StartWorker(*tsk, cfg); err != nil {
			// return err
			logr.Fatal("worker server encountered and error", zap.Error(err))
		}
	}()

	// create and start server
	svr := server.New(cfg, logr, dataStore)
	if err := svr.Start(ctx, diContainer); err != nil {
		return err
	}

	return nil
}
