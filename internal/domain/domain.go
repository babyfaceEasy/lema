package domain

import (
	"context"
	"time"
)

type CommitService interface {
	GetTopCommitAuthors(ctx context.Context, owner, name string, limit int) ([]CommitAuthor, error)
	GetCommitsByRepositoryName(ctx context.Context, owner, name string) ([]Commit, error)
	LoadCommits(ctx context.Context, owner string, name string) error
	GetLatestCommits(ctx context.Context, owner string, name string) error
	GetLatestCommitsNew(ctx context.Context, owner string, name string) error
	ResetCommits(ctx context.Context, owner string, name string) error
}

type RepositoryService interface {
	GetAllRepositories(ctx context.Context) ([]Repository, error)
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
	SaveRepository(ctx context.Context, ownerName string, repoName string, startTime *time.Time) error
	UpdateRepositorySinceDate(ctx context.Context, ownerName string, repoName string, startTime time.Time) error
	UpdateRepositoryStartDate(ctx context.Context, ownerName string, repoName string, startTime time.Time) error
}
