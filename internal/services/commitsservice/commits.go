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

// LoadCommits calls the github client to get the repository commits.
func (cs *commitService) LoadCommits(ctx context.Context, ownerName, repoName string) error {
	logr := cs.logger.With(zap.String("method", "LoadCommits"))

	repoDetails, err := cs.repositoryService.GetRepository(ctx, ownerName, repoName)
	if err != nil {
		return err
	}

	if repoDetails == nil {
		return fmt.Errorf("repository with full name %s/%s does not exist in our system", ownerName, repoName)
	}

	// Call github client for the commits and save in the DB.
	// Get all commits from GitHub; no since date since repo is new.
	commitResponses, err := cs.githubClient.GetCommits(repoName, ownerName, nil, repoDetails.UntilDate)
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
				OwnerName:           ownerName,
				Description:         repoDetails.Description,
				URL:                 repoDetails.URL,
				ProgrammingLanguage: repoDetails.ProgrammingLanguage,
				ForksCount:          repoDetails.ForksCount,
				StarsCount:          repoDetails.StarsCount,
				WatchersCount:       repoDetails.WatchersCount,
				OpenIssuesCount:     repoDetails.OpenIssuesCount,
				UntilDate:           repoDetails.UntilDate,
				SinceDate:           time.Now(), // Set the initial SinceDate to now.
				CreatedAt:           time.Now(),
			},
			Author: domain.Author{
				Name:  com.Commit.Author.Name,
				Email: com.Commit.Author.Email,
			},
			Date:      com.Commit.Author.Date,
			CreatedAt: time.Now(),
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
				UntilDate:           repoDetails.UntilDate,
				CreatedAt:           repoDetails.CreatedAt,
			},
			Author: domain.Author{
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

func (cs *commitService) GetLatestCommitsNew(ctx context.Context) error {
	logr := cs.logger.With(zap.String("method", "GetLatestCommits"))

	repos, err := cs.repositoryService.GetAllRepositories(ctx)
	if err != nil {
		return err
	}

	for _, repoDetails := range repos {

		commitResponses, err := cs.githubClient.GetCommits(repoDetails.Name, repoDetails.OwnerName, &repoDetails.SinceDate, repoDetails.UntilDate)
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
				Date:      com.Commit.Author.Date,
				CreatedAt: time.Now(),
			}
			commitsToUpsert = append(commitsToUpsert, commit)
		}

		// If there are new or updated commits, upsert them.
		if len(commitsToUpsert) > 0 {
			err = cs.commitRepo.UpsertCommits(ctx, commitsToUpsert)
			if err != nil {
				logr.Error("failed to update repository since date", zap.Error(err))
				continue
			}
			// Update the repository's SinceDate to the current time.
			err = cs.repositoryService.UpdateRepositorySinceDate(ctx, repoDetails.OwnerName, repoDetails.Name, time.Now())
			if err != nil {
				// log and move on
				logr.Error("failed to update repository since date", zap.Error(err))
				continue
			}
		}

	}

	return nil
}

func (cs *commitService) ResetCommits(ctx context.Context, ownerName, repoName string) error {
	logr := cs.logger.With(zap.String("method", "ResetCommits"))
	repoDets, err := cs.repositoryService.GetRepository(ctx, ownerName, repoName)
	if err != nil {
		return err
	}

	if err := cs.commitRepo.DeleteCommitsByRepositoryID(ctx, uint(repoDets.ID)); err != nil {
		return err
	}

	if err := tasks.CallLoadCommitsTask(repoDets.OwnerName, repoDets.Name); err != nil {
		return err
	}

	logr.Debug("reset collection for", zap.String("repo_name", repoDets.Name))
	return nil
}
