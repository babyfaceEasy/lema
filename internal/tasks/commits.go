package tasks

import (
	"context"
	"net/http"
	"time"

	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func (t *Task) HandleCommitsUpdateTask(ctx context.Context, a *asynq.Task) error {
	// Get repositories by name
	repos, err := t.store.Repositories.GetAll(context.Background())
	if err != nil {
		t.logger.Error("err: ", zap.Error(err))
		return err
	}

	// Create a GitHub client.
	githubClient := githubapi.NewClient(t.config.GithubBaseUrl, &http.Client{Timeout: 10 * time.Second}, t.logger)

	for _, repoDetails := range repos {

		// Call GitHub API to get new/updated commits since repoDetails.SinceDate up to untilTime.
		commitResponses, err := githubClient.GetCommits(repoDetails.Name, repoDetails.Name, &repoDetails.SinceDate, repoDetails.UntilDate)
		if err != nil {
			return err
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

		t.logger.Info("commits to Upsert", zap.Int("count", len(commitsToUpsert)))

		// If there are new or updated commits, upsert them.
		if len(commitsToUpsert) > 0 {
			err = t.store.Commits.UpsertCommits(context.Background(), commitsToUpsert)
			if err != nil {
				return err
			}
			// Update the repository's SinceDate to the current time.
			err = t.store.Repositories.UpdateSinceDate(context.Background(), repoDetails.Name, time.Now())
			if err != nil {
				t.logger.Error("failed to update repository since date", zap.Error(err))
				return err
			}
		}

	}
	return nil
}
