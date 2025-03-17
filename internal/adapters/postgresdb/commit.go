package postgresdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/babyfaceeasy/lema/internal/domain"
	"github.com/babyfaceeasy/lema/internal/repositories"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type commitStore struct {
	db *sqlx.DB
}

func NewCommitStore(db *sql.DB) repositories.CommitRepository {
	return &commitStore{db: sqlx.NewDb(db, "postgres")}
}

// getOrCreateRepository checks if a repository exists based on its Name Field.
func (s *commitStore) getOrCreateRepository(ctx context.Context, tx *sqlx.Tx, repo *domain.Repository) (int, error) {
	if repo.UID == uuid.Nil {
		repo.UID = uuid.New()
	}

	var id int
	query := `SELECT id FROM repositories WHERE name = $1 LIMIT 1`
	err := tx.GetContext(ctx, &id, query, repo.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			now := time.Now()
			if repo.CreatedAt.IsZero() {
				repo.CreatedAt = now
			}
			if repo.SinceDate.IsZero() {
				repo.SinceDate = now
			}

			insertQuery := `
				INSERT INTO repositories 
				(uid, name, owner_name, description, url, programming_language, forks_count, stars_count, watchers_count, open_issues_count, since_date, created_at, until_date)
				VALUES 
				(:uid, :name, :owner_name, :description, :url, :programming_language, :forks_count, :stars_count, :watchers_count, :open_issues_count, :since_date, :created_at, :until_date)
				RETURNING id
			`
			stmt, err := tx.PrepareNamedContext(ctx, insertQuery)
			if err != nil {
				return 0, fmt.Errorf("failed to prepare named statement: %w", err)
			}
			defer stmt.Close()
			err = stmt.Get(&id, repo)
			if err != nil {
				return 0, fmt.Errorf("failed to insert repository: %w", err)
			}
		} else {
			return 0, fmt.Errorf("failed to check repository existence: %w", err)
		}
	}
	return id, nil
}

// getOrCreateAuthor checks if an author exists based on its Name Field.
func (s *commitStore) getOrCreateAuthor(ctx context.Context, tx *sqlx.Tx, author *domain.Author) (int, error) {
	if author.UID == uuid.Nil {
		author.UID = uuid.New()
	}

	var id int
	query := `SELECT id FROM authors WHERE email = $1 LIMIT 1`
	err := tx.GetContext(ctx, &id, query, author.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			insertQuery := `
				INSERT INTO authors 
				(uid, name, email)
				VALUES 
				(:uid, :name, :email)
				RETURNING id
			`

			stmt, err := tx.PrepareNamedContext(ctx, insertQuery)
			if err != nil {
				return 0, fmt.Errorf("failed to prepare named statement: %w", err)
			}
			defer stmt.Close()
			err = stmt.Get(&id, author)
			if err != nil {
				return 0, fmt.Errorf("failed to insert author: %w", err)
			}
		} else {
			return 0, fmt.Errorf("failed to check author existence: %w", err)
		}
	}
	return id, nil
}

// StoreCommits inserts a list of commits into the database.
func (s *commitStore) StoreCommits(ctx context.Context, commits []domain.Commit) error {
	if len(commits) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	commitQuery := `
        INSERT INTO commits 
            (uid, repository_id, author_id, url, sha, message, commit_date, created_at)
        VALUES 
            (:uid, :repository_id, :author_id, :url, :sha, :message, :commit_date, :created_at)
    `
	for _, commit := range commits {
		repoID, err := s.getOrCreateRepository(ctx, tx, &commit.Repository)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("processing commit insert repo %v: %w", commit.URL, err)
		}
		commit.RepositoryID = repoID

		authorID, err := s.getOrCreateAuthor(ctx, tx, &commit.Author)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("processing commit insert author %v: %w", commit.URL, err)
		}
		commit.AuthorID = authorID

		// Set default values.
		if commit.UID == uuid.Nil {
			commit.UID = uuid.New()
		}
		now := time.Now()
		if commit.CreatedAt.IsZero() {
			commit.CreatedAt = now
		}

		_, err = tx.NamedExecContext(ctx, commitQuery, commit)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("inserting commit %v: %w", commit.URL, err)
		}
	}

	return tx.Commit()
}

// GetCommitsByRepositoryName returns all commits for the repository with the given name.
func (s *commitStore) GetCommitsByRepositoryName(ctx context.Context, repositoryName string) ([]domain.Commit, error) {
	var commits []domain.Commit

	query := `
	SELECT 
		c.id,
		c.uid,
		c.repository_id,
		c.author_id,
		c.url,
		c.sha,
		c.message,
		c.commit_date,
		c.created_at,
		-- c.updated_at,
		-- Repository fields with "Repository." prefix
		r.id AS "Repository.id",
		r.uid AS "Repository.uid",
		r.name AS "Repository.name",
		r.owner_name AS "Repository.owner_name",
		r.description AS "Repository.description",
		r.url AS "Repository.url",
		r.programming_language AS "Repository.programming_language",
		r.forks_count AS "Repository.forks_count",
		r.stars_count AS "Repository.stars_count",
		r.watchers_count AS "Repository.watchers_count",
		r.open_issues_count AS "Repository.open_issues_count",
		r.since_date AS "Repository.since_date",
		r.until_date AS "Repository.until_date",
		r.created_at AS "Repository.created_at",
		-- r.updated_at AS "Repository.updated_at",
		-- Author fields with "Author." prefix
		a.id AS "Author.id",
		a.uid AS "Author.uid",
		a.name AS "Author.name",
		a.email AS "Author.email"
	FROM commits c
	JOIN repositories r ON c.repository_id = r.id
	JOIN authors a ON c.author_id = a.id
	WHERE r.name = $1
	ORDER BY c.commit_date DESC
	`

	err := s.db.SelectContext(ctx, &commits, query, repositoryName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commits for repository %s: %w", repositoryName, err)
	}

	return commits, nil
}

// GetTopCommitAuthors returns the top N commit authors by commit count.
func (s *commitStore) GetTopCommitAuthors(ctx context.Context, limit int) ([]domain.CommitAuthor, error) {
	var authors []domain.CommitAuthor

	query := `
		SELECT 
			a.id,
			a.uid,
			a.name,
			a.email,
			COUNT(c.id) AS commit_count
		FROM authors a
		JOIN commits c ON c.author_id = a.id
		GROUP BY a.id, a.uid, a.name, a.email
		ORDER BY commit_count DESC
		LIMIT $1
	`

	if err := s.db.SelectContext(ctx, &authors, query, limit); err != nil {
		return nil, fmt.Errorf("failed to fetch top commit authors: %w", err)
	}

	return authors, nil
}

// UpsertCommits inserts or updates a slice of commits into the database.
func (s *commitStore) UpsertCommits(ctx context.Context, commits []domain.Commit) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	query := `
		INSERT INTO commits 
			(uid, repository_id, author_id, sha, url, message, commit_date, created_at)
		VALUES 
			(:uid, :repository_id, :author_id, :sha, :url, :message, :commit_date, :created_at)
		ON CONFLICT (repository_id, sha) DO UPDATE SET 
			url = EXCLUDED.url,
			message = EXCLUDED.message,
			date = EXCLUDED.date
	`

	for _, commit := range commits {
		if commit.RepositoryID == 0 {
			repoID, err := s.getOrCreateRepository(ctx, tx, &commit.Repository)
			if err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("upserting commit %s: %w", commit.SHA, err)
			}
			commit.RepositoryID = repoID
		}

		if commit.AuthorID == 0 {
			authorID, err := s.getOrCreateAuthor(ctx, tx, &commit.Author)
			if err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("upserting commit %s: %w", commit.SHA, err)
			}
			commit.AuthorID = authorID
		}

		if commit.UID == uuid.Nil {
			commit.UID = uuid.New()
		}

		if commit.CreatedAt.IsZero() {
			commit.CreatedAt = time.Now()
		}

		if _, err := tx.NamedExecContext(ctx, query, commit); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upserting commit %s: %w", commit.SHA, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *commitStore) DeleteCommitsByRepositoryID(ctx context.Context, repositoryID uint) error {
	query := `
        DELETE FROM commits
        WHERE repository_id = $1;
    `
	if _, err := s.db.ExecContext(ctx, query, repositoryID); err != nil {
		return fmt.Errorf("failed to delete commits: %w", err)
	}
	return nil
}
