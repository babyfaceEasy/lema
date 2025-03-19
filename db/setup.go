package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/babyfaceeasy/lema/config"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

func NewPostgresDb(conf *config.Config) (*sql.DB, error) {
	dsn := conf.DatabaseUrl()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Println("password is", conf.DatabaseUrl())
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

func NewRedisDb(conf *config.Config) (*redis.Client, error) {
	opt, err := redis.ParseURL(conf.GetRedisURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}
	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

func RunMigrations(db *sql.DB, logger *zap.Logger) error {
	if err := goose.Up(db, "db/migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
