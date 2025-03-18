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

// tempConfig to load configs solely
type tempConfig struct {
	// Server
	ApiServerPort string `env:"APISERVER_PORT"`
	ApiServerHost string `env:"APISERVER_HOST"`

	// Database
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

	// App
	AppName       string `env:"APP_NAME"`
	AppEnv        Env    `env:"APP_ENV" envDefault:"dev"`
	ProjectRoot   string `env:"PROJECT_ROOT"`
	CorsWhiteList string `env:"CORS_WHITELIST"`

	// Github
	GithubBaseUrl string `env:"GITHUB_BASE_URL"`
	GithubToken   string `env:"GITHUB_TOKEN"`
}

type Config struct {
	// server
	apiServerPort string `env:"APISERVER_PORT"`
	apiServerHost string `env:"APISERVER_HOST"`

	// DB
	databaseName     string `env:"DB_NAME"`
	databaseHost     string `env:"DB_HOST"`
	databaseUser     string `env:"DB_USER"`
	databasePassword string `env:"DB_PASSWORD"`
	databasePort     string `env:"DB_PORT"`

	// Test DB
	databasePortTest string `env:"DB_PORT_TEST"`

	// Redis
	redisURL  string `env:"REDIS_URL"`
	redisPort string `env:"REDIS_PORT"`
	redisHost string `env:"REDIS_HOST"`

	// app
	appName       string `env:"APP_NAME"`
	appEnv        Env    `env:"APP_ENV" envDefault:"dev"`
	projectRoot   string `env:"PROJECT_ROOT"`
	corsWhiteList string `env:"CORS_WHITELIST"`

	// Github
	githubBaseUrl string `env:"GITHUB_BASE_URL"`
	githubToken   string `env:"GITHUB_TOKEN"`
}

func LoadConfig() (*Config, error) {
	var tc tempConfig
	if err := env.Parse(&tc); err != nil {
		return nil, err
	}

	return &Config{
		// server
		apiServerPort: tc.ApiServerPort,
		apiServerHost: tc.ApiServerHost,

		// DB
		databaseName:     tc.DatabaseName,
		databaseHost:     tc.DatabaseHost,
		databaseUser:     tc.DatabaseUser,
		databasePassword: tc.DatabasePassword,
		databasePort:     tc.DatabasePort,

		// Test DB
		databasePortTest: tc.DatabasePortTest,

		// Redis
		redisURL:  tc.RedisURL,
		redisPort: tc.RedisPort,
		redisHost: tc.RedisHost,

		// App
		appName:       tc.AppName,
		appEnv:        tc.AppEnv,
		projectRoot:   tc.ProjectRoot,
		corsWhiteList: tc.CorsWhiteList,

		// Github
		githubBaseUrl: tc.GithubBaseUrl,
		githubToken:   tc.GithubToken,
	}, nil
}

func (c *Config) GetApiServerPort() string {
	return c.apiServerPort
}

func (c *Config) GetApiServerHost() string {
	return c.apiServerHost
}

func (c *Config) GetDatabaseName() string {
	return c.databaseName
}

func (c *Config) GetDatabaseHost() string {
	return c.databaseHost
}

func (c *Config) GetDatabaseUser() string {
	return c.databaseUser
}

func (c *Config) GetDatabasePassword() string {
	return c.databasePassword
}

func (c *Config) GetDatabasePort() string {
	return c.databasePort
}

func (c *Config) GetDatabasePortTest() string {
	return c.databasePortTest
}

func (c *Config) GetRedisURL() string {
	return c.redisURL
}

func (c *Config) GetRedisPort() string {
	return c.redisPort
}

func (c *Config) GetRedisHost() string {
	return c.redisHost
}

func (c *Config) GetAppName() string {
	return c.appName
}

func (c *Config) GetAppEnv() Env {
	return c.appEnv
}

func (c *Config) GetProjectRoot() string {
	return c.projectRoot
}

func (c *Config) GetCorsWhiteList() string {
	return c.corsWhiteList
}

func (c *Config) GetGithubBaseUrl() string {
	return c.githubBaseUrl
}

func (c *Config) GetGithubToken() string {
	return c.githubToken
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
	port := c.GetDatabasePort()

	if c.appEnv == Env_Test {
		port = c.GetDatabasePortTest()
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.GetDatabaseUser(),
		c.GetDatabasePassword(),
		c.GetDatabaseHost(),
		port,
		c.GetDatabaseName(),
	)
}

func (c *Config) RedisAddress() string {
	return fmt.Sprintf("%s:%s", c.GetRedisHost(), c.GetRedisPort())
}
