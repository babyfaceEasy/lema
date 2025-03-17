package repositories

import (
	"context"
	"time"

	"github.com/babyfaceeasy/lema/internal/domain"
)

type RepositoryRepository interface {
	CreateOrUpdate(ctx context.Context, repo domain.Repository) error
	ByName(ctx context.Context, owner, name string) (*domain.Repository, error)
	UpdateSinceDate(ctx context.Context, owner string, name string, newSinceDate time.Time) error
	UpdateStartDate(ctx context.Context, owner string, name string, newStartDate time.Time) error
	Exists(ctx context.Context, owner, name string) (bool, error)
	GetAll(ctx context.Context) ([]domain.Repository, error)
}
