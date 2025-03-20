package server

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/container"
	"github.com/babyfaceeasy/lema/internal/routes"
	"github.com/babyfaceeasy/lema/internal/store"
	"go.uber.org/zap"
)

type ApiServer struct {
	config *config.Config
	logger *zap.Logger
	store  *store.Store
}

func New(config *config.Config, logger *zap.Logger, store *store.Store) *ApiServer {
	return &ApiServer{config: config, logger: logger, store: store}
}

func (s *ApiServer) Start(ctx context.Context, diContainer *container.Container) error {
	mux := routes.RegisterRoutes(
		s.config, s.logger, 
		s.store, 
		diContainer.GetCommitService(), 
		diContainer.GetRepositoryService(), 
		diContainer.GetTaskQueue(),
	)
	server := &http.Server{
		Addr:    net.JoinHostPort(s.config.GetApiServerHost(), s.config.GetApiServerPort()),
		Handler: mux,
	}

	// handle start up server
	go func() {
		s.logger.Info("apiserver running", zap.String("port", s.config.GetApiServerHost()))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("apiserver failed to listen and serve", zap.Error(err))
		}
	}()

	// handle the shutdown logic
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("failed to shutdown server gracefully", zap.Error(err))
		}
	}()

	wg.Wait()
	return nil
}
