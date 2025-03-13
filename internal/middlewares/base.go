package middlewares

import (
	"github.com/babyfaceeasy/lema/config"
	"go.uber.org/zap"
)

type Middleware struct {
	config *config.Config
	logger *zap.Logger
}

func New(config *config.Config, logger *zap.Logger) *Middleware {
	return &Middleware{config: config, logger: logger}
}
