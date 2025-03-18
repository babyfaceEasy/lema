package routes

import (
	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/handlers"
	"github.com/babyfaceeasy/lema/internal/middlewares"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var (
	handler    *handlers.Handler
	middleware *middlewares.Middleware
)

func RegisterRoutes(
	config *config.Config,
	logger *zap.Logger,
	store *store.Store,
	commitSvc domain.CommitService,
	repositorySvc domain.RepositoryService,
) *mux.Router {
	router := mux.NewRouter()

	handler = handlers.New(config, logger, store, commitSvc, repositorySvc)
	middleware = middlewares.New(config, logger)

	// global middlewares
	router.Use(middleware.CORS)
	router.Use(middleware.LoggerMiddleware)

	// v1 endpoints
	apiV1 := router.PathPrefix("/v1").Subrouter()
	apiV1.HandleFunc("", handler.Ping).Methods("GET")
	apiV1.HandleFunc("/repositories/{repository_name}", handler.GetRepository).Methods("GET")
	apiV1.HandleFunc("/repositories/{repository_name}/commits", handler.GetRepositoryCommits).Methods("GET")
	apiV1.HandleFunc("/repositories/monitor", handler.MonitorRepository).Methods("POST")
	apiV1.HandleFunc("/repositories/reset-collection", handler.ResetCollection).Methods("POST")
	// commits
	apiV1.HandleFunc("/commit-authors/top", handler.GetTopCommitAuthors).Methods("GET")

	return router
}
