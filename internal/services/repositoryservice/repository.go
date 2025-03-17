package repositoryservice

import (
	"context"
	"fmt"
	"time"

	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"github.com/babyfaceeasy/lema/internal/repositories"
	"go.uber.org/zap"
)

type repositoryService struct {
	logger         *zap.Logger
	repoRepository repositories.RepositoryRepository
	githubClient   *githubapi.Client
}

func NewRepositoryService(logger *zap.Logger, repoRepository repositories.RepositoryRepository, gitHubService *githubapi.Client) domain.RepositoryService {
	logger = logger.With(zap.String("package", "repositoryservice"))
	return &repositoryService{
		logger:         logger,
		repoRepository: repoRepository,
		githubClient:   gitHubService,
	}
}

func (rs *repositoryService) GetAllRepositories(ctx context.Context) ([]domain.Repository, error) {
	logr := rs.logger.With(zap.String("method", "GetAllRepositories"))
	repos, err := rs.repoRepository.GetAll(ctx)
	if err != nil {
		logr.Error("error in getting all repositories from DB", zap.Error(err))
		return nil, err
	}

	logr.Debug("total repositories in the DB", zap.Int("repos_count", len(repos)))
	return repos, nil
}

func (rs *repositoryService) GetRepository(ctx context.Context, owner, repo string) (*domain.Repository, error) {
	logr := rs.logger.With(zap.String("method", "GetRepository"))

	repoDetails, err := rs.repoRepository.ByName(ctx, owner, repo)
	if err != nil {
		logr.Error("An error in getting details of a given repository", zap.Error(err), zap.String("repo_name", repo))
	}

	return repoDetails, nil
}

func (rs *repositoryService) SaveRepository(ctx context.Context, owner string, repo string, startTime *time.Time) error {
	logr := rs.logger.With(zap.String("method", "SaveRepository"))

	exists, err := rs.repoRepository.Exists(ctx, owner, repo)
	if err != nil {
		return err
	}

	if exists {
		logr.Info("repository exists already", zap.String("repo_name", repo), zap.String("owner_name", owner))
		return fmt.Errorf("repository name : %s/%s already in our system", owner, repo)
	}

	// try getting the details from github client
	repoDetails, err := rs.githubClient.GetRepositoryDetails(repo, owner)
	if err != nil {
		logr.Error("error in getting repository details", zap.Error(err))
	}

	newRepo := domain.Repository{
		Name:                repoDetails.Name,
		OwnerName:           owner,
		Description:         repoDetails.Description,
		URL:                 repoDetails.URL,
		ProgrammingLanguage: repoDetails.ProgrammingLanguage,
		ForksCount:          repoDetails.ForksCount,
		StarsCount:          repoDetails.StarsCount,
		WatchersCount:       repoDetails.WatchersCount,
		OpenIssuesCount:     repoDetails.OpenIssuesCount,
		UntilDate:           startTime,
		SinceDate:           time.Now(), // Set the initial SinceDate to now.
		CreatedAt:           time.Now(),
	}

	if err := rs.repoRepository.CreateOrUpdate(ctx, newRepo); err != nil {
		logr.Error("error in saving repository", zap.Error(err))
		return err
	}

	logr.Info("repository with name was saved successfully", zap.String("repo_name", repo))

	// todo: create a task to start getting of the commits for the given repository, make use of the repository ID here
	return nil
}

func (rs *repositoryService) UpdateRepositorySinceDate(ctx context.Context, owner string, repo string, sinceTime time.Time) error {
	logr := rs.logger.With(zap.String("method", "UpdateRepositorySinceDate"))

	if err := rs.repoRepository.UpdateSinceDate(ctx, owner, repo, sinceTime); err != nil {
		logr.Error("error in updating repository update since date")
		return fmt.Errorf("error in updating repository since date: %w", err)
	}

	return nil
}

func (rs *repositoryService) UpdateRepositoryStartDate(ctx context.Context, ownerName string, repoName string, startTime time.Time) error {
	logr := rs.logger.With(zap.String("method", "UpdateRepositoryStartDate"))

	if err := rs.repoRepository.UpdateStartDate(ctx, ownerName, repoName, startTime); err != nil {
		logr.Error("error in updating repository update start date")
		return fmt.Errorf("error in updating repository start date: %w", err)
	}

	return nil
}
