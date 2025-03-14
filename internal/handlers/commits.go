package handlers

import (
	"net/http"
	"strconv"

	"github.com/babyfaceeasy/lema/internal/utils"
	"go.uber.org/zap"
)

func (h Handler) GetTopCommitAuthors(w http.ResponseWriter, r *http.Request) {
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
