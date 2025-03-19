package postgresdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/repositories"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repositoryStore struct {
	db *sqlx.DB
}

func NewRepositoryStore(db *sql.DB) repositories.RepositoryRepository {
	return &repositoryStore{db: sqlx.NewDb(db, "postgres")}
}

// ByName returns the repository details if it exists in the system.
func (s *repositoryStore) ByName(ctx context.Context, ownerName, repoName string) (*domain.Repository, error) {
	const query = `SELECT * FROM repositories WHERE name ILIKE $1 AND owner_name ILIKE $2`

	var repo domain.Repository
	if err := s.db.GetContext(ctx, &repo, query, repoName, ownerName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("not found name = %s and owner_name = %s\n", repoName, ownerName)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find repository by name and owner: %w", err)
	}

	return &repo, nil
}

// GetAll returns all the repositories stored in our DB.
func (s *repositoryStore) GetAll(ctx context.Context) ([]domain.Repository, error) {
	const query = `SELECT * FROM repositories`

	var repos []domain.Repository

	err := s.db.SelectContext(ctx, &repos, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories from DB : %w", err)
	}

	return repos, nil
}

// Exists returns true if a repository with the given name exists.
func (s *repositoryStore) Exists(ctx context.Context, owner, name string) (bool, error) {
	var exists bool
	query := `SELECT exists(SELECT 1 FROM repositories WHERE name = $1 and owner_name = $2)`
	err := s.db.GetContext(ctx, &exists, query, name, owner)
	if err != nil {
		return false, fmt.Errorf("failed to check repository existence: %w", err)
	}
	return exists, nil
}

// UpdateSinceDate updates the since_date (and updated_at) for the repository with the given name.
func (s *repositoryStore) UpdateSinceDate(ctx context.Context, repositoryName string, ownerName string, newSinceDate time.Time) error {
	query := `
		UPDATE repositories 
		SET since_date = $1
		WHERE name = $2 and owner_name = $3
	`
	_, err := s.db.ExecContext(ctx, query, newSinceDate, repositoryName, ownerName)
	if err != nil {
		return fmt.Errorf("failed to update since_date for repository %s: %w", repositoryName, err)
	}
	return nil
}

// CreateOrUpdate creates a new repository record if one does not exist (by name),
// or updates the existing record with the provided fields.
func (s *repositoryStore) CreateOrUpdateOLD(ctx context.Context, repo domain.Repository) error {
	log.Printf("repo ben passed :%+v\n", repo)
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
					(uid, name, owner_name, description, url, programming_language, forks_count, stars_count, watchers_count, open_issues_count, created_at, since_date, until_date)
				VALUES 
					(:uid, :name, :owner_name, :description, :url, :programming_language, :forks_count, :stars_count, :watchers_count, :open_issues_count, :created_at, :since_date, :until_date)
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

// CreateOrUpdate creates a new repository record if one does not exist (by name),
// or updates the existing record with the provided fields.
func (s *repositoryStore) CreateOrUpdate(ctx context.Context, repo domain.Repository) error {
	log.Printf("repo being passed: %+v\n", repo)
	var id int
	// Check if repository exists using its name.
	err := s.db.GetContext(ctx, &id, `
		SELECT id FROM repositories 
		WHERE name = $1 and owner_name = $2
		LIMIT 1
	`, repo.Name, repo.OwnerName)
	if err != nil {
		if err == sql.ErrNoRows {
			// Repository does not exist: insert a new one.
			if repo.UID == uuid.Nil {
				repo.UID = uuid.New()
			}
			now := time.Now()
			if repo.CreatedAt.IsZero() {
				repo.CreatedAt = now
			}
			/*
				if repo.SinceDate.IsZero() {
					repo.SinceDate = now
				}
			*/
			insertQuery := `
				INSERT INTO repositories 
					(uid, name, owner_name, description, url, programming_language, forks_count, stars_count, watchers_count, open_issues_count, created_at, since_date, until_date)
				VALUES 
					(:uid, :name, :owner_name, :description, :url, :programming_language, :forks_count, :stars_count, :watchers_count, :open_issues_count, :created_at, :since_date, :until_date)
				RETURNING id
			`
			stmt, err := s.db.PrepareNamedContext(ctx, insertQuery)
			if err != nil {
				return fmt.Errorf("failed to prepare named statement for repository insert: %w", err)
			}
			defer stmt.Close()
			err = stmt.Get(&id, repo)
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
				since_date = :since_date,
				until_date = :until_date
			WHERE name = :name and owner_name = :owner_name
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

// UpdateStartDate updates the until_date field.
func (s *repositoryStore) UpdateStartDate(ctx context.Context, ownerName string, repositoryName string, newStartDate time.Time) error {
	var dateParam interface{}
	if newStartDate.IsZero() {
		// If newStartDate is zero, set dateParam to nil so that until_date becomes NULL in Postgres.
		dateParam = nil
	} else {
		dateParam = newStartDate
	}
	query := `
		UPDATE repositories 
		SET until_date = $1
		WHERE name = $2 and owner_name = $3
	`
	_, err := s.db.ExecContext(ctx, query, dateParam, repositoryName, ownerName)
	if err != nil {
		return fmt.Errorf("failed to update since_date for repository %s: %w", repositoryName, err)
	}
	return nil
}
