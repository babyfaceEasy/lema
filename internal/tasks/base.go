package tasks

import (
	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/store"
	"go.uber.org/zap"
)

type Task struct {
	config            *config.Config
	logger            *zap.Logger
	store             *store.Store
	commitService     domain.CommitService
	repositoryService domain.RepositoryService
}

func New(config *config.Config, logger *zap.Logger, store *store.Store, commitSvc domain.CommitService, repoSvc domain.RepositoryService) *Task {
	logger = logger.With(zap.String("package", "tasks"))
	return &Task{config: config, logger: logger, store: store, repositoryService: repoSvc, commitService: commitSvc}
}
