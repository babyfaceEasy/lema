package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Env string

const (
	Env_Test = "test"
	Env_Dev  = "dev"
)

type Config struct {
	// server
	ApiServerPort string `env:"APISERVER_PORT"`
	ApiServerHost string `env:"APISERVER_HOST"`

	// DB
	DatabaseName     string `env:"DB_NAME"`
	DatabaseHost     string `env:"DB_HOST"`
	DatabaseUser     string `env:"DB_USER"`
	DatabasePassword string `env:"DB_PASSWORD"`
	DatabasePort     string `env:"DB_PORT"`

	// Test DB
	DatabasePortTest string `env:"DB_PORT_TEST"`

	// Redis
	RedisURL  string `env:"REDIS_URL"`
	RedisPort string `env:"REDIS_PORT"`
	RedisHost string `env:"REDIS_HOST"`

	// app
	AppName       string `env:"APP_NAME"`
	AppEnv        Env    `env:"APP_ENV" envDefault:"dev"`
	ProjectRoot   string `env:"PROJECT_ROOT"`
	CorsWhiteList string `env:"CORS_WHITELIST"`

	// Github
	GithubBaseUrl string `env:"GITHUB_BASE_URL"`
	GithubToken   string `env:"GITHUB_TOKEN"`
}

func New() (*Config, error) {
	var cfg Config
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return &cfg, fmt.Errorf("failed to load config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) DatabaseUrl() string {
	port := c.DatabasePort

	if c.AppEnv == Env_Test {
		port = c.DatabasePortTest
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseHost,
		port,
		c.DatabaseName,
	)
}

func (c *Config) RedisAddress() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}
