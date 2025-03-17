package handlers

import (
	"net/http"
	"strconv"

	"github.com/babyfaceeasy/lema/internal/messages"
	"github.com/babyfaceeasy/lema/internal/utils"
	"go.uber.org/zap"
)

func (h Handler) GetTopCommitAuthorsOLD(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		// Set a default value if "limit" is not provided.
		limitStr = "10"
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		h.logger.Error("invalid limit value: %v", zap.Error(err))

		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: "Invalid request",
			Error:   []string{"Invalid limit value"},
		})
		utils.SendResponse(w, code, res)
		return
	}
	h.logger.Info("value of query parameters passed", zap.Int("limit", limit))

	authors, err := h.store.Commits.GetTopCommitAuthors(r.Context(), limit)
	if err != nil {
	}

	code, res := h.response(http.StatusOK, ResponseFormat{
		Status:  true,
		Message: "top author commits retrieved successfully",
		Data:    authors,
	})
	utils.SendResponse(w, code, res)
}

func (h Handler) GetTopCommitAuthors(w http.ResponseWriter, r *http.Request) {
	logr := h.logger.With(zap.String("method", "GetTopCommitAuthors"))

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "10"
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		logr.Error("invalid limit value: %v", zap.Error(err))

		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: "Invalid request",
			Error:   []string{"Invalid limit value"},
		})
		utils.SendResponse(w, code, res)
		return
	}
	logr.Info("value of query parameters passed", zap.Int("limit", limit))

	authors, err := h.commitService.GetTopCommitAuthors(r.Context(), "chronuim", "chronuim", limit)
	if err != nil {
		logr.Error("an error occurred", zap.Error(err))

		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
			Error:   []string{"Invalid limit value"},
		})
		utils.SendResponse(w, code, res)
		return
	}

	code, res := h.response(http.StatusOK, ResponseFormat{
		Status:  true,
		Message: "top author commits retrieved successfully",
		Data:    authors,
	})
	utils.SendResponse(w, code, res)
}
