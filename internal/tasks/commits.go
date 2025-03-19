package tasks

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"github.com/babyfaceeasy/lema/internal/store"
	"github.com/bytedance/sonic"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type LoadCommitsTaskInput struct {
	RepositoryName  string
	RepositoryOwner string
}

type GetLatestCommitsTaskInput struct {
	RepositoryName  string
	RepositoryOwner string
}

type ResetCommitsTaskInput struct {
	RepositoryName  string
	RepositoryOwner string
}

// cron handlers
func (t *Task) HandleCommitsUpdateTaskOLD(ctx context.Context, a *asynq.Task) error {
	// Get repositories by name
	repos, err := t.store.Repositories.GetAll(context.Background())
	if err != nil {
		t.logger.Error("err: ", zap.Error(err))
		return err
	}

	// Create a GitHub client.
	githubClient := githubapi.NewClient(t.config.GetGithubBaseUrl(), &http.Client{Timeout: 10 * time.Second}, t.logger, t.config)

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

func (t *Task) HandleCommitsUpdateTask(ctx context.Context, a *asynq.Task) error {
	logr := t.logger.With(zap.String("method", "HandleCommitsUpdateTask"))

	repos, err := t.repositoryService.GetAllRepositories(ctx)
	if err != nil {
		return err
	}

	for _, repoDetails := range repos {
		err := CallLatestCommitsTask(repoDetails.OwnerName, repoDetails.Name)
		if err != nil {
			logr.Error("error in adding repositories to get latest task", zap.Error(err))
		}

		logr.Debug("added repo for getting latest commits", zap.String("repo_name", repoDetails.Name))
	}

	return nil
}

func CallLoadCommitsTask(owner, name string) error {
	i := LoadCommitsTaskInput{RepositoryOwner: owner, RepositoryName: name}
	payload, err := sonic.Marshal(i)
	if err != nil {
		return err
	}

	info, err := client.Enqueue(asynq.NewTask("ops:load_commits", payload), asynq.Retention(5*time.Hour), asynq.Queue(TypeQueueCritical))
	if err != nil {
		return err
	}

	log.Printf(" [*] Successfully enqueued task: %+v", *info)

	return nil
}

// task handlers
func (t *Task) HandleLoadCommitsTask(ctx context.Context, a *asynq.Task) error {
	var p LoadCommitsTaskInput
	if err := sonic.Unmarshal(a.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return t.commitService.LoadCommits(ctx, p.RepositoryOwner, p.RepositoryName)
}

func CallLatestCommitsTask(owner, name string) error {
	i := LoadCommitsTaskInput{RepositoryOwner: owner, RepositoryName: name}
	payload, err := sonic.Marshal(i)
	if err != nil {
		return err
	}

	info, err := client.Enqueue(asynq.NewTask("ops:latest_commits", payload), asynq.Retention(5*time.Hour), asynq.Queue(TypeQueueDefault))
	if err != nil {
		return err
	}

	log.Printf(" [*] Successfully enqueued task: %s\n", info.Type)

	return nil
}

func (t *Task) HandleLatestCommitsTask(ctx context.Context, a *asynq.Task) error {
	// logr := t.logger.With(zap.String("method", "HandleLatestCommitsTask"))
	var p GetLatestCommitsTaskInput
	if err := sonic.Unmarshal(a.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	// return t.commitService.GetLatestCommits(ctx, p.RepositoryOwner, p.RepositoryName)
	return t.commitService.GetLatestCommitsNew(ctx, p.RepositoryOwner, p.RepositoryName)
}

func CallResetCommitsTask(owner, name string) error {
	i := ResetCommitsTaskInput{RepositoryOwner: owner, RepositoryName: name}
	payload, err := sonic.Marshal(i)
	if err != nil {
		return err
	}

	info, err := client.Enqueue(asynq.NewTask("ops:reset_commits", payload), asynq.Retention(5*time.Hour), asynq.Queue(TypeQueueDefault))
	if err != nil {
		return err
	}

	log.Printf(" [*] Successfully enqueued task: %s\n", info.Type)

	return nil
}

func (t *Task) HandleResetCommitsTask(ctx context.Context, a *asynq.Task) error {
	// logr := t.logger.With(zap.String("method", "HandleResetCommitsTask"))
	var p ResetCommitsTaskInput
	if err := sonic.Unmarshal(a.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return t.commitService.ResetCommits(ctx, p.RepositoryOwner, p.RepositoryName)
}
