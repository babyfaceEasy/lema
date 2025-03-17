package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"github.com/babyfaceeasy/lema/internal/messages"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/babyfaceeasy/lema/internal/tasks"
	"github.com/babyfaceeasy/lema/internal/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (h Handler) GetRepositoryCommitsOLD(w http.ResponseWriter, r *http.Request) {
	repositoryName := mux.Vars(r)["repository_name"]
	ownerName := r.URL.Query().Get("owner_name")
	if ownerName == "" {
		ownerName = repositoryName
	}

	h.logger.Info("repository name passed", zap.String("repo_name", repositoryName))
	h.logger.Info("owner name passed", zap.String("owner_name", ownerName))

	// Parse the "until" query parameter.
	var untilTime *time.Time
	untilStr := r.URL.Query().Get("until")
	if untilStr != "" {
		parsedUntil, err := time.Parse(time.RFC3339, untilStr)
		if err != nil {
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: "Invalid 'until' date format, must be RFC3339",
			})
			utils.SendResponse(w, code, res)
			return
		}
		now := time.Now().UTC()
		// If the parsed "until" is before now, we override it to now.
		if parsedUntil.Before(now) {
			parsedUntil = now
		}
		untilTime = &parsedUntil
	}

	// Create a GitHub client.
	githubClient := githubapi.NewClient(h.config.GithubBaseUrl, &http.Client{Timeout: 10 * time.Second}, h.logger, h.config)

	// Check if the repository exists in the DB.
	exists, err := h.store.Repositories.Exists(r.Context(), repositoryName)
	if err != nil {
		h.logger.Error("error in checking if repository exists", zap.Error(err))
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	if exists {
		// Get repository details from DB (which includes SinceDate).
		repoDetails, err := h.store.Repositories.ByName(r.Context(), repositoryName)
		if err != nil {
			h.logger.Error("error in getting repo details from db", zap.Error(err))
			code, res := h.response(http.StatusNotFound, ResponseFormat{
				Status:  false,
				Message: messages.SomethingWentWrong,
			})
			utils.SendResponse(w, code, res)
			return
		}

		/*
			// Get stored commits from DB.
			dbCommits, err := h.store.Commits.GetCommitsByRepositoryName(r.Context(), repositoryName)
			if err != nil {
				h.logger.Error("error in getting commits belonging to a repository", zap.String("repository_name", repositoryName), zap.Error(err))
				code, res := h.response(http.StatusBadRequest, ResponseFormat{
					Status:  false,
					Message: messages.SomethingWentWrong,
				})
				utils.SendResponse(w, code, res)
				return
			}
		*/

		// Call GitHub API to get new/updated commits since repoDetails.SinceDate up to untilTime.
		commitResponses, err := githubClient.GetCommits(repositoryName, ownerName, &repoDetails.SinceDate, untilTime)
		if err != nil {
			h.logger.Error("error in getting commits from github client", zap.Error(err))
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: messages.SomethingWentWrong,
			})
			utils.SendResponse(w, code, res)
			return
		}

		// Get the latest "until" date.
		// If an untilDate is passed, use it as the new untilDate else use the one from repoDetails.
		latestUntilDate := untilTime
		if latestUntilDate == nil {
			latestUntilDate = repoDetails.UntilDate
		}

		// Convert GitHub commit responses to store.Commit objects.
		var commitsToUpsert []store.Commit
		for _, com := range commitResponses {
			commit := store.Commit{
				SHA:     com.SHA,
				URL:     com.URL,
				Message: com.Commit.Message,
				Repository: store.Repository{
					// Use the existing repository details.
					Name:                repoDetails.Name,
					OwnerName:           repoDetails.OwnerName,
					Description:         repoDetails.Description,
					URL:                 repoDetails.URL,
					ProgrammingLanguage: repoDetails.ProgrammingLanguage,
					ForksCount:          repoDetails.ForksCount,
					StarsCount:          repoDetails.StarsCount,
					WatchersCount:       repoDetails.WatchersCount,
					OpenIssuesCount:     repoDetails.OpenIssuesCount,
					SinceDate:           repoDetails.SinceDate,
					UntilDate:           latestUntilDate,
					CreatedAt:           repoDetails.CreatedAt,
				},
				Author: store.Author{
					Name:  com.Commit.Author.Name,
					Email: com.Commit.Author.Email,
				},
				Date:      com.Commit.Author.Date,
				CreatedAt: time.Now(),
			}
			commitsToUpsert = append(commitsToUpsert, commit)
		}

		// If there are new or updated commits, upsert them.
		if len(commitsToUpsert) > 0 {
			err = h.store.Commits.UpsertCommits(r.Context(), commitsToUpsert)
			if err != nil {
				h.logger.Error("error in performing upsert operation", zap.Error(err))
				code, res := h.response(http.StatusBadRequest, ResponseFormat{
					Status:  false,
					Message: messages.SomethingWentWrong,
				})
				utils.SendResponse(w, code, res)
				return
			}
			// Update the repository's SinceDate to the current time.
			err = h.store.Repositories.UpdateSinceDate(r.Context(), repositoryName, time.Now())
			if err != nil {
				h.logger.Error("failed to update repository since date", zap.Error(err))
			}
		}

		// Retrieve and return the (updated) stored commits.
		dbCommits, err := h.store.Commits.GetCommitsByRepositoryName(r.Context(), repositoryName)
		if err != nil {
			h.logger.Error("error in getting commits belonging to a repository", zap.String("repository_name", repositoryName), zap.Error(err))
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: messages.SomethingWentWrong,
			})
			utils.SendResponse(w, code, res)
			return
		}
		code, res := h.response(http.StatusOK, ResponseFormat{
			Status:  true,
			Message: "Commits retrieved successfully",
			Data:    dbCommits,
		})
		utils.SendResponse(w, code, res)
		return
	}

	// Repository does not exist in DB.
	// Get repository details from GitHub.
	repoResponse, err := githubClient.GetRepositoryDetails(repositoryName, ownerName)
	if err != nil {
		h.logger.Error("error in getting repository details from github server", zap.Error(err))
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: err.Error(),
		})
		utils.SendResponse(w, code, res)
		return
	}

	// Get all commits from GitHub; no since date since repo is new.
	commitResponses, err := githubClient.GetCommits(repositoryName, ownerName, nil, untilTime)
	if err != nil {
		h.logger.Error("error in getting commits from github server", zap.Error(err))
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: err.Error(),
		})
		utils.SendResponse(w, code, res)
		return
	}

	// Build commit list to store.
	var commits []store.Commit
	for _, com := range commitResponses {
		newCommit := store.Commit{
			SHA:     com.SHA,
			URL:     com.URL,
			Message: com.Commit.Message,
			Repository: store.Repository{
				Name:                repoResponse.Name,
				OwnerName:           repoResponse.Owner.Login,
				Description:         repoResponse.Description,
				URL:                 repoResponse.URL,
				ProgrammingLanguage: repoResponse.ProgrammingLanguage,
				ForksCount:          repoResponse.ForksCount,
				StarsCount:          repoResponse.StarsCount,
				WatchersCount:       repoResponse.WatchersCount,
				OpenIssuesCount:     repoResponse.OpenIssuesCount,
				UntilDate:           untilTime,
				SinceDate:           time.Now(), // Set the initial SinceDate to now.
				CreatedAt:           time.Now(),
			},
			Author: store.Author{
				Name:  com.Commit.Author.Name,
				Email: com.Commit.Author.Email,
			},
			Date:      com.Commit.Author.Date,
			CreatedAt: time.Now(),
		}
		commits = append(commits, newCommit)
	}

	// Store all commits for the new repository.
	err = h.store.Commits.StoreCommits(r.Context(), commits)
	if err != nil {
		h.logger.Error("error in storing all commits at a go", zap.Error(err))
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: messages.SomethingWentWrong,
		})
		utils.SendResponse(w, code, res)
		return
	}

	/*
		// Update repository details with current SinceDate.
		repoDetails.SinceDate = time.Now()
		err = h.store.Repositories.CreateOrUpdate(r.Context(), repoDetails)
		if err != nil {
			h.logger.Error("failed to create or update repository", zap.Error(err))
		}
	*/

	// Retrieve and return the stored commits.
	storedCommits, err := h.store.Commits.GetCommitsByRepositoryName(r.Context(), repoResponse.Name)
	if err != nil {
		h.logger.Error("error in getting stored commits", zap.Error(err))
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
		Data:    storedCommits,
	})
	utils.SendResponse(w, code, res)
}

func (h Handler) GetRepositoryCommits(w http.ResponseWriter, r *http.Request) {
	logr := h.logger.With(zap.String("method", "GetRepositoryCommits"))

	repositoryName := mux.Vars(r)["repository_name"]
	ownerName := r.URL.Query().Get("owner_name")
	if ownerName == "" {
		ownerName = repositoryName
	}

	logr.Debug("repository name passed", zap.String("repo_name", repositoryName))
	logr.Debug("owner name passed", zap.String("owner_name", ownerName))

	// Retrieve and return the stored commits.
	storedCommits, err := h.commitService.GetCommitsByRepositoryName(r.Context(), ownerName, repositoryName)
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
		Data:    storedCommits,
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
