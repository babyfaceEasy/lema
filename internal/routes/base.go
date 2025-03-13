package routes

import (
	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/handlers"
	"github.com/babyfaceeasy/lema/internal/middlewares"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

var (
	handler    *handlers.Handler
	middleware *middlewares.Middleware
)

func RegisterRoutes(config *config.Config, logger *zap.Logger, store *store.Store) *mux.Router {
	router := mux.NewRouter()

	handler = handlers.New(config, logger, store)
	middleware = middlewares.New(config, logger)

	// Swagger endpoint
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// global middlewares
	router.Use(middleware.LoggerMiddleware)

	// v1 endpoints
	apiV1 := router.PathPrefix("/v1").Subrouter()
	apiV1.HandleFunc("", handler.Ping).Methods("GET")

	// commit routes
	registerCommitRoutes(apiV1)
	registerRepositoryRoutes(apiV1)

	return router
}
