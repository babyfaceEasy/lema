package handlers

import (
	"net/http"
	"time"

	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"github.com/babyfaceeasy/lema/internal/messages"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/babyfaceeasy/lema/internal/utils"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (h Handler) GetRepositoryCommits(w http.ResponseWriter, r *http.Request) {
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
	githubClient := githubapi.NewClient(h.config.GithubBaseUrl, &http.Client{Timeout: 10 * time.Second}, h.logger)

	// Check if the repository exists in the DB.
	exists, err := h.store.Repositories.Exists(r.Context(), repositoryName)
	if err != nil {
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: err.Error(),
		})
		utils.SendResponse(w, code, res)
		return
	}

	if exists {
		// Get repository details from DB (which includes SinceDate).
		repoDetails, err := h.store.Repositories.ByName(r.Context(), repositoryName)
		if err != nil {
			h.logger.Error("error in getting repo details", zap.Error(err))
			code, res := h.response(http.StatusNotFound, ResponseFormat{
				Status:  false,
				Message: messages.NotFound,
			})
			utils.SendResponse(w, code, res)
			return
		}

		// Get stored commits from DB.
		dbCommits, err := h.store.Commits.GetCommitsByRepositoryName(r.Context(), repositoryName)
		if err != nil {
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: err.Error(),
			})
			utils.SendResponse(w, code, res)
			return
		}

		// Call GitHub API to get new/updated commits since repoDetails.SinceDate up to untilTime.
		commitResponses, err := githubClient.GetCommits(repositoryName, ownerName, &repoDetails.SinceDate, untilTime)
		if err != nil {
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: err.Error(),
			})
			utils.SendResponse(w, code, res)
			return
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
					Description:         repoDetails.Description,
					URL:                 repoDetails.URL,
					ProgrammingLanguage: repoDetails.ProgrammingLanguage,
					ForksCount:          repoDetails.ForksCount,
					StarsCount:          repoDetails.StarsCount,
					WatchersCount:       repoDetails.WatchersCount,
					OpenIssuesCount:     repoDetails.OpenIssuesCount,
					SinceDate:           repoDetails.SinceDate,
					CreatedAt:           repoDetails.CreatedAt,
				},
				Author: store.Author{
					Name:  com.Commit.Author.Name,
					Email: com.Commit.Author.Email,
				},
				// Ideally, use the commit's actual date.
				Date:      time.Now(),
				CreatedAt: time.Now(),
			}
			commitsToUpsert = append(commitsToUpsert, commit)
		}

		// If there are new or updated commits, upsert them.
		if len(commitsToUpsert) > 0 {
			err = h.store.Commits.UpsertCommits(r.Context(), commitsToUpsert)
			if err != nil {
				code, res := h.response(http.StatusBadRequest, ResponseFormat{
					Status:  false,
					Message: err.Error(),
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
		dbCommits, err = h.store.Commits.GetCommitsByRepositoryName(r.Context(), repositoryName)
		if err != nil {
			code, res := h.response(http.StatusBadRequest, ResponseFormat{
				Status:  false,
				Message: err.Error(),
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
				Description:         repoResponse.Description,
				URL:                 repoResponse.URL,
				ProgrammingLanguage: repoResponse.ProgrammingLanguage,
				ForksCount:          repoResponse.ForksCount,
				StarsCount:          repoResponse.StarsCount,
				WatchersCount:       repoResponse.WatchersCount,
				OpenIssuesCount:     repoResponse.OpenIssuesCount,
				// Set the initial SinceDate to now.
				SinceDate: time.Now(),
				CreatedAt: time.Now(),
			},
			Author: store.Author{
				Name:  com.Commit.Author.Name,
				Email: com.Commit.Author.Email,
			},
			Date:      time.Now(), // Replace with actual commit date if available.
			CreatedAt: time.Now(),
		}
		commits = append(commits, newCommit)
	}

	// Store all commits for the new repository.
	err = h.store.Commits.StoreCommits(r.Context(), commits)
	if err != nil {
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: err.Error(),
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
		code, res := h.response(http.StatusBadRequest, ResponseFormat{
			Status:  false,
			Message: err.Error(),
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
