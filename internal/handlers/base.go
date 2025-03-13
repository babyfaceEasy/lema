package handlers

import (
	"net/http"

	"github.com/babyfaceeasy/lema/config"
	"github.com/babyfaceeasy/lema/internal/messages"
	"github.com/babyfaceeasy/lema/internal/store"
	"go.uber.org/zap"
)

type Handler struct {
	config *config.Config
	logger *zap.Logger
	store  *store.Store
}

func New(config *config.Config, logger *zap.Logger, store *store.Store) *Handler {
	return &Handler{config: config, logger: logger, store: store}
}

type ResponseFormat struct {
	Status  bool     `json:"status"`
	Data    any      `json:"data,omitempty"`
	Error   []string `json:"error,omitempty"`
	Message string   `json:"message"`
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
