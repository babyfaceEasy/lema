package domain

import (
	"context"
	"time"

	"github.com/babyfaceeasy/lema/pkg/pagination"
)

type CommitService interface {
	GetTopCommitAuthors(ctx context.Context, owner, name string, limit int) ([]CommitAuthor, error)
	GetCommitsByRepositoryName(ctx context.Context, owner, name string, page, pageSize int) ([]Commit, *pagination.Pagination, error)
	LoadCommits(ctx context.Context, owner string, name string) error
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
