package githubservice

import (
	"context"
	"time"

	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/integrations/githubapi"
	"go.uber.org/zap"
)

type GitHubService interface {
	GetRepositoryDetails(repositoryName, ownerName string) (*domain.Repository, error)
	GetCommitsNew(ctx context.Context, repositoryName, ownerName string, since, until *time.Time, pageSize int, commitCh chan<- domain.Commit) error
}

type githubService struct {
	client *githubapi.Client
	logger *zap.Logger
}

// NewService creates a new GitHubService.
func NewGithubService(client *githubapi.Client, logger *zap.Logger) GitHubService {
	logger = logger.With(zap.String("package", "githubservice"))
	return &githubService{
		client: client,
		logger: logger,
	}
}

// GetRepositoryDetails calls the underlying client's GetRepositoryDetails
func (s *githubService) GetRepositoryDetails(repositoryName, ownerName string) (*domain.Repository, error) {
	repoResp, err := s.client.GetRepositoryDetails(repositoryName, ownerName)
	if err != nil {
		return nil, err
	}

	domainRepo := domain.Repository{
		Name:                repoResp.Name,
		OwnerName:           repoResp.Owner.Login,
		Description:         repoResp.Description,
		URL:                 repoResp.URL,
		ProgrammingLanguage: repoResp.ProgrammingLanguage,
		ForksCount:          repoResp.ForksCount,
		StarsCount:          repoResp.StarsCount,
		WatchersCount:       repoResp.WatchersCount,
		OpenIssuesCount:     repoResp.OpenIssuesCount,
	}
	return &domainRepo, nil
}

func (s *githubService) GetCommitsNew(ctx context.Context, repositoryName, ownerName string, since, until *time.Time, pageSize int, commitCh chan<- domain.Commit) error {
	// Create a temporary channel for commit responses from the client.
	tempCh := make(chan githubapi.CommitResponse, 200)
	// Launch the client's GetCommitsNew concurrently.
	go func() {
		defer close(tempCh)
		err := s.client.GetCommitsNew(ctx, repositoryName, ownerName, since, until, pageSize, tempCh)
		if err != nil {
			s.logger.Error("Error fetching commits", zap.Error(err))
		}
	}()

	// Convert each githubapi.CommitResponse to domain.Commit and send it.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case cr, ok := <-tempCh:
			if !ok {
				// Channel is closed; we're done.
				return nil
			}
			dc := convertToDomainCommit(cr)
			commitCh <- dc
		}
	}
}

// convertToDomainCommit converts a githubapi.CommitResponse to a domain.Commit.
func convertToDomainCommit(cr githubapi.CommitResponse) domain.Commit {
	return domain.Commit{
		SHA:        cr.SHA,
		URL:        cr.URL,
		Message:    cr.Commit.Message,
		CommitDate: cr.Commit.Author.Date,
		CreatedAt:  time.Now(),
		Author: domain.Author{
			Name:  cr.Commit.Author.Name,
			Email: cr.Commit.Author.Email,
		},
		// The Repository field might be set later in the commit service.
	}
}
