package commitsservice

import (
	"context"
	"fmt"
	"time"

	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"github.com/babyfaceeasy/lema/internal/repositories"
	"github.com/babyfaceeasy/lema/internal/tasks"
	"go.uber.org/zap"
)

type commitService struct {
	githubClient      *githubapi.Client
	logger            *zap.Logger
	commitRepo        repositories.CommitRepository
	repositoryService domain.RepositoryService
}

func NewCommitService(gitHubService *githubapi.Client, commitRepo repositories.CommitRepository, logger *zap.Logger, repoSvc domain.RepositoryService) domain.CommitService {
	logger = logger.With(zap.String("package", "commitservice"))
	return &commitService{
		githubClient:      gitHubService,
		commitRepo:        commitRepo,
		logger:            logger,
		repositoryService: repoSvc,
	}
}

func (cs *commitService) GetTopCommitAuthors(ctx context.Context, owner, name string, limit int) ([]domain.CommitAuthor, error) {
	logr := cs.logger.With(zap.String("method", "GetTopCommitAuthors"))

	authors, err := cs.commitRepo.GetTopCommitAuthors(ctx, limit)
	if err != nil {
		logr.Error("error in getting GetTopCommitAuthors", zap.Error(err))
	}
	logr.Info("Fetched top commit authors")
	return authors, nil
}

func (cs *commitService) GetCommitsByRepositoryName(ctx context.Context, owner, name string) ([]domain.Commit, error) {
	logr := cs.logger.With(zap.String("method", "GetCommitsByRepositoryName"))

	commits, err := cs.commitRepo.GetCommitsByRepositoryName(ctx, name)
	if err != nil {
		logr.Error("error in GetCommitsByRepositoryName", zap.Error(err))
		return nil, err
	}
	return commits, nil
}

func (cs *commitService) LoadCommitsOLD(ctx context.Context, ownerName, repoName string) error {
	logr := cs.logger.With(zap.String("method", "LoadCommits"))

	repoDetails, err := cs.repositoryService.GetRepository(ctx, ownerName, repoName)
	if err != nil {
		return err
	}

	if repoDetails == nil {
		return fmt.Errorf("repository with full name %s/%s does not exist in our system", ownerName, repoName)
	}

	var untilTime *time.Time
	if !repoDetails.UntilDate.IsZero() {
		untilTime = repoDetails.UntilDate
	}
	commitResponses, err := cs.githubClient.GetCommits(repoName, ownerName, nil, untilTime)
	if err != nil {
		return err
	}

	// Build commit list to store.
	var commits []domain.Commit
	for _, com := range commitResponses {
		newCommit := domain.Commit{
			SHA:     com.SHA,
			URL:     com.URL,
			Message: com.Commit.Message,
			Repository: domain.Repository{
				Name:                repoDetails.Name,
				OwnerName:           repoDetails.OwnerName,
				Description:         repoDetails.Description,
				URL:                 repoDetails.URL,
				ProgrammingLanguage: repoDetails.ProgrammingLanguage,
				ForksCount:          repoDetails.ForksCount,
				StarsCount:          repoDetails.StarsCount,
				WatchersCount:       repoDetails.WatchersCount,
				OpenIssuesCount:     repoDetails.OpenIssuesCount,
				UntilDate:           repoDetails.UntilDate,
				SinceDate:           time.Now(),
				CreatedAt:           time.Now(),
			},
			Author: domain.Author{
				Name:  com.Commit.Author.Name,
				Email: com.Commit.Author.Email,
			},
			CommitDate: com.Commit.Author.Date,
			CreatedAt:  time.Now(),
		}
		commits = append(commits, newCommit)
	}

	// Store all commits for the new repository.
	err = cs.commitRepo.StoreCommits(ctx, commits)
	if err != nil {
		return err
	}

	logr.Info("loaded commits for repository successfully", zap.String("owner_name", repoDetails.OwnerName), zap.String("repo_name", repoDetails.Name))

	return nil
}

func (cs *commitService) LoadCommits(ctx context.Context, ownerName, repoName string) error {
	logr := cs.logger.With(zap.String("method", "LoadCommits"))

	repoDetails, err := cs.repositoryService.GetRepository(ctx, ownerName, repoName)
	if err != nil {
		return err
	}
	if repoDetails == nil {
		return fmt.Errorf("repository %s/%s does not exist in our system", ownerName, repoName)
	}

	// Create a buffered channel for commit responses.
	commitCh := make(chan githubapi.CommitResponse, 200)

	// Launch the concurrent GitHub client to fetch commits.
	go func() {
		// Ensure the channel is closed when done.
		defer close(commitCh)
		err := cs.githubClient.GetCommitsNew(ctx, repoName, ownerName, nil, repoDetails.UntilDate, 100, commitCh)
		if err != nil {
			logr.Error("Error fetching commits", zap.Error(err))
		} else {
			logr.Info("Finished fetching commits from GitHub")
		}
	}()

	var commits []domain.Commit
	batchSize := 50
	commitCount := 0

	// For-select loop to process incoming commits.
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while processing commits")
		case commitResp, ok := <-commitCh:
			if !ok {
				// Channel closed: save any remaining commits.
				logr.Info("Commit channel closed", zap.Int("totalCommitsReceived", commitCount))
				if len(commits) > 0 {
					if err := cs.commitRepo.StoreCommits(ctx, commits); err != nil {
						return fmt.Errorf("failed to store remaining commits: %w", err)
					}
					logr.Info("Stored final batch of commits", zap.Int("batchSize", len(commits)))
				}
				logr.Info("Loaded all commits successfully", zap.Int("totalCommitsSaved", commitCount))
				return nil
			}

			commitCount++
			logr.Debug("Received commit", zap.String("sha", commitResp.SHA))
			// Convert CommitResponse to domain.Commit.
			newCommit := domain.Commit{
				SHA:     commitResp.SHA,
				URL:     commitResp.URL,
				Message: commitResp.Commit.Message,
				Author: domain.Author{
					Name:  commitResp.Commit.Author.Name,
					Email: commitResp.Commit.Author.Email,
				},
				CommitDate: commitResp.Commit.Author.Date,
				CreatedAt:  time.Now(),
				Repository: domain.Repository{
					Name:                repoDetails.Name,
					OwnerName:           repoDetails.OwnerName,
					Description:         repoDetails.Description,
					URL:                 repoDetails.URL,
					ProgrammingLanguage: repoDetails.ProgrammingLanguage,
					ForksCount:          repoDetails.ForksCount,
					StarsCount:          repoDetails.StarsCount,
					WatchersCount:       repoDetails.WatchersCount,
					OpenIssuesCount:     repoDetails.OpenIssuesCount,
					UntilDate:           repoDetails.UntilDate,
					SinceDate:           time.Now(),
					CreatedAt:           time.Now(),
				},
			}

			commits = append(commits, newCommit)
			// If batch is full, save to DB and reset the slice.
			if len(commits) >= batchSize {
				logr.Info("Storing batch of commits", zap.Int("batchSize", len(commits)))
				if err := cs.commitRepo.StoreCommits(ctx, commits); err != nil {
					return fmt.Errorf("failed to store commit batch: %w", err)
				}
				commits = commits[:0]
			}
		}
	}
}

func (cs *commitService) GetLatestCommits(ctx context.Context, ownerName, repoName string) error {
	logr := cs.logger.With(zap.String("method", "GetLatestCommits"))

	repoDetails, err := cs.repositoryService.GetRepository(ctx, ownerName, repoName)
	if err != nil {
		return err
	}

	if repoDetails == nil {
		return fmt.Errorf("repository with full name %s/%s does not exist in our system", ownerName, repoName)
	}

	commitResponses, err := cs.githubClient.GetCommits(repoName, ownerName, &repoDetails.SinceDate, repoDetails.UntilDate)
	if err != nil {
		return err
	}

	// Convert GitHub commit responses to store.Commit objects.
	var commitsToUpsert []domain.Commit
	for _, com := range commitResponses {
		commit := domain.Commit{
			SHA:     com.SHA,
			URL:     com.URL,
			Message: com.Commit.Message,
			Repository: domain.Repository{
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
				UntilDate:           repoDetails.UntilDate,
				CreatedAt:           repoDetails.CreatedAt,
			},
			Author: domain.Author{
				Name:  com.Commit.Author.Name,
				Email: com.Commit.Author.Email,
			},
			CommitDate: com.Commit.Author.Date,
			CreatedAt:  time.Now(),
		}
		commitsToUpsert = append(commitsToUpsert, commit)
	}

	if len(commitsToUpsert) > 0 {
		err = cs.commitRepo.UpsertCommits(ctx, commitsToUpsert)
		if err != nil {
			return err
		}
		// Update the repository's SinceDate to the current time.
		err = cs.repositoryService.UpdateRepositorySinceDate(ctx, repoDetails.OwnerName, repoDetails.Name, time.Now())
		if err != nil {
			logr.Error("failed to update repository since date", zap.Error(err))
			return err
		}
	}

	return nil
}

func (cs *commitService) GetLatestCommitsNew(ctx context.Context, ownerName, repoName string) error {
	logr := cs.logger.With(zap.String("method", "GetLatestCommits"))

	repoDetails, err := cs.repositoryService.GetRepository(ctx, ownerName, repoName)
	if err != nil {
		return err
	}
	if repoDetails == nil {
		return fmt.Errorf("repository %s/%s does not exist in our system", ownerName, repoName)
	}

	// Create a buffered channel for commit responses.
	commitCh := make(chan githubapi.CommitResponse, 200)

	// Launch the concurrent GitHub client to fetch commits.
	go func() {
		// Ensure the channel is closed when done.
		defer close(commitCh)
		err := cs.githubClient.GetCommitsNew(ctx, repoName, ownerName, &repoDetails.SinceDate, repoDetails.UntilDate, 100, commitCh)
		if err != nil {
			logr.Error("Error fetching commits", zap.Error(err))
		} else {
			logr.Info("Finished fetching commits from GitHub")
		}
	}()

	var commits []domain.Commit
	batchSize := 50
	commitCount := 0

	// For-select loop to process incoming commits.
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while processing commits")
		case commitResp, ok := <-commitCh:
			if !ok {
				// Channel closed: save any remaining commits.
				logr.Info("Commit channel closed", zap.Int("totalCommitsReceived", commitCount))
				if len(commits) > 0 {
					if err := cs.commitRepo.StoreCommits(ctx, commits); err != nil {
						return fmt.Errorf("failed to store remaining commits: %w", err)
					}
					logr.Info("Stored final batch of commits", zap.Int("batchSize", len(commits)))
				}
				logr.Info("Loaded all commits successfully", zap.Int("totalCommitsSaved", commitCount))
				return nil
			}

			commitCount++
			logr.Debug("Received commit", zap.String("sha", commitResp.SHA))
			// Convert CommitResponse to domain.Commit.
			newCommit := domain.Commit{
				SHA:     commitResp.SHA,
				URL:     commitResp.URL,
				Message: commitResp.Commit.Message,
				Author: domain.Author{
					Name:  commitResp.Commit.Author.Name,
					Email: commitResp.Commit.Author.Email,
				},
				CommitDate: commitResp.Commit.Author.Date,
				CreatedAt:  time.Now(),
				Repository: domain.Repository{
					Name:                repoDetails.Name,
					OwnerName:           repoDetails.OwnerName,
					Description:         repoDetails.Description,
					URL:                 repoDetails.URL,
					ProgrammingLanguage: repoDetails.ProgrammingLanguage,
					ForksCount:          repoDetails.ForksCount,
					StarsCount:          repoDetails.StarsCount,
					WatchersCount:       repoDetails.WatchersCount,
					OpenIssuesCount:     repoDetails.OpenIssuesCount,
					UntilDate:           repoDetails.UntilDate,
					SinceDate:           repoDetails.SinceDate,
					CreatedAt:           repoDetails.CreatedAt,
				},
			}

			commits = append(commits, newCommit)
			// If batch is full, save to DB and reset the slice.
			if len(commits) >= batchSize {
				logr.Info("Upserting batch of commits", zap.Int("batchSize", len(commits)))
				if err := cs.commitRepo.UpsertCommits(ctx, commits); err != nil {
					return fmt.Errorf("failed to upsert commit batch: %w", err)
				}
				commits = commits[:0]

				// Update the repository's SinceDate to the current time.
				err = cs.repositoryService.UpdateRepositorySinceDate(ctx, repoDetails.OwnerName, repoDetails.Name, time.Now())
				if err != nil {
					logr.Error("failed to update the since_date for repo with name", zap.String("repo_name", repoDetails.Name), zap.Error(err))
					return fmt.Errorf("failed to update the since_date for repo with ID: %s  cos: %w", repoDetails.UID, err)
				}
			}
		}
	}
}

func (cs *commitService) ResetCommits(ctx context.Context, ownerName, repoName string) error {
	logr := cs.logger.With(zap.String("method", "ResetCommits"))
	repoDetails, err := cs.repositoryService.GetRepository(ctx, ownerName, repoName)
	if err != nil {
		return err
	}

	if err := cs.commitRepo.DeleteCommitsByRepositoryID(ctx, uint(repoDetails.ID)); err != nil {
		return err
	}

	if err := tasks.CallLoadCommitsTask(repoDetails.OwnerName, repoDetails.Name); err != nil {
		return err
	}

	logr.Debug("reset collection for", zap.String("repo_name", repoDetails.Name))
	return nil
}
