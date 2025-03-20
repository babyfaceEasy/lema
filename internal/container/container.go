package container

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/db"
	"github.com/babyfaceeasy/lema/internal/adapters/postgresdb"
	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"github.com/babyfaceeasy/lema/internal/queue"
	"github.com/babyfaceeasy/lema/internal/services/commitsservice"
	"github.com/babyfaceeasy/lema/internal/services/githubservice"
	"github.com/babyfaceeasy/lema/internal/services/repositoryservice"
	"go.uber.org/zap"
)

type Container struct {
	config            *config.Config
	dbConn            *sql.DB
	commitService     domain.CommitService
	repositoryService domain.RepositoryService
	taskQueue         queue.TaskQueue
}

func NewContainer(config *config.Config, logger *zap.Logger) *Container {
	// db
	dbConn, err := db.NewPostgresDb(config)
	if err != nil {
		logger.Panic("Error connecting to database", zap.Error(err))
	}

	// Repositories
	commitRepo := postgresdb.NewCommitStore(dbConn)
	repositoryRepo := postgresdb.NewRepositoryStore(dbConn)

	// Clients
	githubClient := githubapi.NewClient(config.GetGithubBaseUrl(), &http.Client{Timeout: 10 * time.Second}, logger, config)

	// Services
	githubSvc := githubservice.NewGithubService(githubClient, logger)
	repositorySvc := repositoryservice.NewRepositoryService(logger, repositoryRepo, githubSvc)
	commitSvc := commitsservice.NewCommitService(githubSvc, commitRepo, logger, repositorySvc)

	// Queue
	inMemQueue := queue.NewInMemoryQueue(5, 100, logger, commitSvc, repositorySvc)

	return &Container{
		config:            config,
		dbConn:            dbConn,
		commitService:     commitSvc,
		repositoryService: repositorySvc,
		taskQueue:         inMemQueue,
	}
}

func (c *Container) GetCommitService() domain.CommitService {
	return c.commitService
}

func (c *Container) GetRepositoryService() domain.RepositoryService {
	return c.repositoryService
}

func (c *Container) GetTaskQueue() queue.TaskQueue {
	return c.taskQueue
}

func (c *Container) Close() {
	c.dbConn.Close()
}
