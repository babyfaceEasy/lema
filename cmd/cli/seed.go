package main

import (
	"log"
	"os"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/db"
	"github.com/babyfaceeasy/lema/pkg/logger"
	"github.com/babyfaceeasy/lema/pkg/seeder"
	"go.uber.org/zap"
)

func main() {
	// load configurations
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("failed to initialize: %v ", err)
	}

	// logger
	logr, err := logger.NewLogger(string(cfg.GetAppEnv()))
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logr.Sync()

	dbConn, err := db.NewPostgresDb(cfg)
	if err != nil {
		logr.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbConn.Close()

	if err := seeder.Seed(dbConn); err != nil {
		logr.Fatal("seeding failed", zap.Error(err))
	}

	logr.Info("Seeder finished successfully")
	os.Exit(0)
}
