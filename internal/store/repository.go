package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type RepositoryStore struct {
	db *sqlx.DB
}

func NewRepositoryStore(db *sql.DB) *RepositoryStore {
	return &RepositoryStore{db: sqlx.NewDb(db, "postgres")}
}

type Repository struct {
	ID                  int        `db:"id" json:"-"`
	UID                 uuid.UUID  `db:"uid" json:"id,omitempty"`
	Name                string     `db:"name" json:"name"`
	OwnerName           string     `db:"owner_name" json:"owner_name"`
	Description         string     `db:"description" json:"description"`
	URL                 string     `db:"url" json:"url"`
	ProgrammingLanguage string     `db:"programming_language" json:"language"`
	ForksCount          int        `db:"forks_count" json:"forks_count"`
	StarsCount          int        `db:"stars_count" json:"stars_count"`
	WatchersCount       int        `db:"watchers_count" json:"watchers_count"`
	OpenIssuesCount     int        `db:"open_issues_count" json:"open_issues_count"`
	UntilDate           *time.Time `db:"until_date" json:"-"`
	SinceDate           time.Time  `db:"since_date" json:"-"`
	CreatedAt           time.Time  `db:"created_at" json:"-"`
}

// ByName returns the repository details if it exists in the system.
func (s *RepositoryStore) ByName(ctx context.Context, repoName string) (*Repository, error) {
	const query = `SELECT * FROM repositories WHERE name = $1`

	var repo Repository
	if err := s.db.GetContext(ctx, &repo, query, repoName); err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return &repo, nil
}

func (s *RepositoryStore) GetAll(ctx context.Context) ([]Repository, error) {
	const query = `SELECT * FROM repositories`

	var repos []Repository

	err := s.db.SelectContext(ctx, &repos, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories from DB : %w", err)
	}

	return repos, nil
}

// Exists returns true if a repository with the given name exists.
func (s *RepositoryStore) Exists(ctx context.Context, name string) (bool, error) {
	var exists bool
	query := `SELECT exists(SELECT 1 FROM repositories WHERE name = $1)`
	err := s.db.GetContext(ctx, &exists, query, name)
	if err != nil {
		return false, fmt.Errorf("failed to check repository existence: %w", err)
	}
	return exists, nil
}

// UpdateSinceDate updates the since_date (and updated_at) for the repository with the given name.
func (s *RepositoryStore) UpdateSinceDate(ctx context.Context, repositoryName string, newSinceDate time.Time) error {
	query := `
		UPDATE repositories 
		SET since_date = $1
		WHERE name = $2
	`
	_, err := s.db.ExecContext(ctx, query, newSinceDate, repositoryName)
	if err != nil {
		return fmt.Errorf("failed to update since_date for repository %s: %w", repositoryName, err)
	}
	return nil
}

// CreateOrUpdate creates a new repository record if one does not exist (by name),
// or updates the existing record with the provided fields.
func (s *RepositoryStore) CreateOrUpdate(ctx context.Context, repo Repository) error {
	var id int
	// Check if repository exists using its name.
	err := s.db.GetContext(ctx, &id, `
		SELECT id FROM repositories 
		WHERE name = $1
		LIMIT 1
	`, repo.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			if repo.UID == uuid.Nil {
				repo.UID = uuid.New()
			}
			now := time.Now()
			if repo.CreatedAt.IsZero() {
				repo.CreatedAt = now
			}
			if repo.SinceDate.IsZero() {
				repo.SinceDate = now
			}
			insertQuery := `
				INSERT INTO repositories 
					(uid, name, owner_name, description, url, programming_language, forks_count, stars_count, watchers_count, open_issues_count, created_at, updated_at, since_date, until_date)
				VALUES 
					(:uid, :name, :owner_name, :description, :url, :programming_language, :forks_count, :stars_count, :watchers_count, :open_issues_count, :created_at, :updated_at, :since_date, :until_date)
				RETURNING id
			`
			err = s.db.GetContext(ctx, &id, insertQuery, repo)
			if err != nil {
				return fmt.Errorf("failed to insert repository %s: %w", repo.Name, err)
			}
		} else {
			return fmt.Errorf("failed to check repository existence for %s: %w", repo.Name, err)
		}
	} else {
		// Repository exists, so update it.
		// If repo.SinceDate is zero, update it to now.
		if repo.SinceDate.IsZero() {
			repo.SinceDate = time.Now()
		}
		updateQuery := `
			UPDATE repositories SET 
				description = :description,
				url = :url,
				owner_name = :owner_name,
				programming_language = :programming_language,
				forks_count = :forks_count,
				stars_count = :stars_count,
				watchers_count = :watchers_count,
				open_issues_count = :open_issues_count,
				updated_at = :updated_at,
				since_date = :since_date
				until_date = :until_date
			WHERE name = :name
		`
		res, err := s.db.NamedExecContext(ctx, updateQuery, repo)
		if err != nil {
			return fmt.Errorf("failed to update repository %s: %w", repo.Name, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected when updating repository %s: %w", repo.Name, err)
		}
		if affected == 0 {
			return fmt.Errorf("no repository updated for %s", repo.Name)
		}
	}
	return nil
}
