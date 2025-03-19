package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/babyfaceeasy/lema/internal/messages"
	"github.com/babyfaceeasy/lema/internal/tasks"
	"github.com/babyfaceeasy/lema/internal/utils"
	"github.com/babyfaceeasy/lema/pkg/pagination"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (h Handler) GetRepositoryCommits(w http.ResponseWriter, r *http.Request) {
	logr := h.logger.With(zap.String("method", "GetRepositoryCommits"))

	repositoryName := mux.Vars(r)["repository_name"]
	ownerName := r.URL.Query().Get("owner_name")
	if ownerName == "" {
		ownerName = repositoryName
	}

	logr.Debug("repository name passed", zap.String("repo_name", repositoryName))
	logr.Debug("owner name passed", zap.String("owner_name", ownerName))

	// pagination parameters
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("page_size"))
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	// Retrieve and return the stored commits.
	storedCommits, pg, err := h.commitService.GetCommitsByRepositoryName(r.Context(), ownerName, repositoryName, page, pageSize)
	if err != nil {
		logr.Error("error in getting stored commits", zap.Error(err))
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	code, res := h.response(http.StatusOK, ResponseFormat{
		Status:  true,
		Message: "Commits stored and retrieved successfully",
		Data:    pagination.PagedResponse{Pagination: pg, Data: storedCommits},
	})
	utils.SendResponse(w, code, res)
}

func (h Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
	logr := h.logger.With(zap.String("method", "GetRepository"))

	repositoryName := mux.Vars(r)["repository_name"]
	ownerName := r.URL.Query().Get("owner_name")
	if ownerName == "" {
		ownerName = repositoryName
	}

	logr.Debug("repository name passed", zap.String("repo_name", repositoryName))
	logr.Debug("owner name passed", zap.String("owner_name", ownerName))

	repositoryDetails, err := h.repositoryService.GetRepository(r.Context(), ownerName, repositoryName)
	if err != nil {
		logr.Error("error in getting repository details", zap.Error(err), zap.String("owner_name", ownerName), zap.String("repo_name", repositoryName))
	}

	if repositoryDetails == nil {
		code, res := h.response(http.StatusNotFound, ResponseFormat{
			Status:  false,
			Message: messages.NotFound,
			Data:    repositoryDetails,
		})
		utils.SendResponse(w, code, res)
		return
	}

	code, res := h.response(http.StatusOK, ResponseFormat{
		Status:  true,
		Message: "Repository Details",
		Data:    repositoryDetails,
	})
	utils.SendResponse(w, code, res)
}

func (h Handler) MonitorRepository(w http.ResponseWriter, r *http.Request) {
	logr := h.logger.With(zap.String("method", "MonitorRepository"))

	var req monitorRepositoryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: "invalid payload request",
		})
		utils.SendResponse(w, code, res)
		return
	}
	defer r.Body.Close()

	// validate
	if err := req.Validate(); err != nil {
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: messages.InvalidRequest,
		})
		if verrs, ok := err.(validation.Errors); ok {
			res.Data = h.withValidationErrors(verrs)
		}
		utils.SendResponse(w, code, res)
		return
	}

	// check the startTime
	if req.StartTimeStr != "" {
		req.StartTime, err = time.Parse(time.RFC3339, req.StartTimeStr)
		if err != nil {
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: "Invalid 'start_time' date format, must be RFC3339",
			})
			utils.SendResponse(w, code, res)
			return
		}
	}

	req.OwnerName = strings.ToLower(req.OwnerName)
	req.RepositoryName = strings.ToLower(req.RepositoryName)

	// Check if user exists
	repoDetails, err := h.repositoryService.GetRepository(r.Context(), req.OwnerName, req.RepositoryName)
	if err != nil {
		logr.Error("error in getting repository:", zap.Error(err))
		code, res := h.response(http.StatusInternalServerError, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	if repoDetails != nil {
		code, res := h.response(http.StatusConflict, ResponseFormat{
			Status:  false,
			Message: fmt.Sprintf("Repository named %s/%s is been monitored already.", req.OwnerName, req.RepositoryName),
		})
		utils.SendResponse(w, code, res)
		return
	}

	// logic
	// todo: work on how to check if it exists already to return 429 error
	if err := h.repositoryService.SaveRepository(r.Context(), req.OwnerName, req.RepositoryName, &req.StartTime); err != nil {
		logr.Error("error in saving repository:", zap.Error(err))
		code, res := h.response(http.StatusInternalServerError, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	repoDetails, err = h.repositoryService.GetRepository(r.Context(), req.OwnerName, req.RepositoryName)
	if err != nil {
		logr.Error("error in getting repository:", zap.Error(err))
		code, res := h.response(http.StatusInternalServerError, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	if err := tasks.CallLoadCommitsTask(repoDetails.OwnerName, repoDetails.Name); err != nil {
		logr.Error("error in creating task to load commits:", zap.Error(err))
		code, res := h.response(http.StatusInternalServerError, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	code, res := h.response(http.StatusOK, ResponseFormat{
		Status:  true,
		Message: fmt.Sprintf("Monitoring started for repository named %s/%s", req.OwnerName, req.RepositoryName),
	})
	utils.SendResponse(w, code, res)
}

func (h Handler) ResetCollection(w http.ResponseWriter, r *http.Request) {
	logr := h.logger.With(zap.String("method", "ResetCollection"))

	var req resetCollectionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: "invalid payload request",
		})
		utils.SendResponse(w, code, res)
		return
	}
	defer r.Body.Close()

	// validate
	if err := req.Validate(); err != nil {
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: messages.InvalidRequest,
		})
		if verrs, ok := err.(validation.Errors); ok {
			res.Data = h.withValidationErrors(verrs)
		}
		utils.SendResponse(w, code, res)
		return
	}

	// check the startTime
	if req.StartTimeStr != "" {
		req.StartTime, err = time.Parse(time.RFC3339, req.StartTimeStr)
		if err != nil {
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: "Invalid 'start_time' date format, must be RFC3339",
			})
			utils.SendResponse(w, code, res)
			return
		}
	}

	repoDetails, err := h.repositoryService.GetRepository(r.Context(), req.OwnerName, req.RepositoryName)
	if err != nil {
		logr.Error("error in getting repository details", zap.String("owner_name", req.OwnerName), zap.String("repo_name", req.RepositoryName), zap.Error(err))
		code, res := h.response(http.StatusInternalServerError, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	if repoDetails == nil {
		code, res := h.response(http.StatusNotFound, ResponseFormat{
			Status:  false,
			Message: messages.NotFound,
		})
		utils.SendResponse(w, code, res)
		return
	}

	if err := h.repositoryService.UpdateRepositoryStartDate(r.Context(), repoDetails.OwnerName, repoDetails.Name, req.StartTime); err != nil {
		logr.Error("error in updating start date", zap.Error(err))
		code, res := h.response(http.StatusInternalServerError, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	if err := tasks.CallResetCommitsTask(repoDetails.OwnerName, repoDetails.Name); err != nil {
		logr.Error("error in initiating the background task for reset commits", zap.Error(err))
		code, res := h.response(http.StatusInternalServerError, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	code, res := h.response(http.StatusOK, ResponseFormat{
		Status:  true,
		Message: fmt.Sprintf("Reset commits started for repository named %s/%s", repoDetails.OwnerName, repoDetails.Name),
	})
	utils.SendResponse(w, code, res)
}
