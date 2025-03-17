package repositories

import (
	"context"

	"github.com/babyfaceeasy/lema/internal/domain"
)

type CommitRepository interface {
	StoreCommits(ctx context.Context, commits []domain.Commit) error
	GetCommitsByRepositoryName(ctx context.Context, name string) ([]domain.Commit, error)
	GetTopCommitAuthors(ctx context.Context, limit int) ([]domain.CommitAuthor, error)
	UpsertCommits(ctx context.Context, commits []domain.Commit) error
	DeleteCommitsByRepositoryID(ctx context.Context, repositoryID uint) error
}
