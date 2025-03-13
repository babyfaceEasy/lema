package store

import "database/sql"

type Store struct {
	Commits      *CommitStore
	Authors      *AuthorStore
	Repositories *RepositoryStore
}

func New(db *sql.DB) *Store {
	return &Store{
		Commits:      NewCommitStore(db),
		Authors:      NewAuthorStore(db),
		Repositories: NewRepositoryStore(db),
	}
}
