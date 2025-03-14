package tasks

import (
	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/store"
	"go.uber.org/zap"
)

type Task struct {
	config *config.Config
	logger *zap.Logger
	store  *store.Store
}

func New(config *config.Config, logger *zap.Logger, store *store.Store) *Task {
	return &Task{config: config, logger: logger, store: store}
}
