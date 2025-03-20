package handlers

import (
	"net/http"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/messages"
	"github.com/babyfaceeasy/lema/internal/queue"
	"github.com/babyfaceeasy/lema/internal/store"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"go.uber.org/zap"
)

type Handler struct {
	config            *config.Config
	logger            *zap.Logger
	store             *store.Store
	commitService     domain.CommitService
	repositoryService domain.RepositoryService
	taskQueue         queue.TaskQueue
}

func New(
	config *config.Config,
	logger *zap.Logger,
	store *store.Store,
	commitService domain.CommitService,
	repositoryService domain.RepositoryService,
	taskQueue queue.TaskQueue,
) *Handler {
	logger = logger.With(zap.String("package", "handlers"))
	return &Handler{config: config, logger: logger, store: store, commitService: commitService, repositoryService: repositoryService, taskQueue: taskQueue}
}

type ResponseFormat struct {
	Status  bool     `json:"status"`
	Data    any      `json:"data,omitempty"`
	Error   []string `json:"error,omitempty"`
	Message string   `json:"message"`
}

func (h Handler) withValidationErrors(errs validation.Errors) map[string]string {
	fieldErrors := make(map[string]string, len(errs))
	for field, err := range errs {
		fieldErrors[field] = err.Error()
	}

	return fieldErrors
}

func (h Handler) response(code int, res ResponseFormat) (int, ResponseFormat) {
	if code == 0 {
		code = 200
	}

	if !res.Status {
		res.Status = code < http.StatusBadRequest
	}

	if res.Message == "" {
		res.Message = messages.OperationWasSuccessful
		if code == http.StatusNotFound {
			res.Message = messages.NotFound
		} else if code >= http.StatusBadRequest {
			res.Message = messages.SomethingWentWrong
		}
	}

	return code, res
}
