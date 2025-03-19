package repositories

import (
	"context"

	"github.com/babyfaceeasy/lema/internal/domain"
)

type CommitRepository interface {
	StoreCommits(ctx context.Context, commits []domain.Commit) error
	GetCommitsByRepositoryName(ctx context.Context, owner, name string, page, pageSize int) ([]domain.Commit, int, error)
	GetTopCommitAuthors(ctx context.Context, limit int) ([]domain.CommitAuthor, error)
	UpsertCommits(ctx context.Context, commits []domain.Commit) error
	DeleteCommitsByRepositoryID(ctx context.Context, repositoryID uint) error
}
