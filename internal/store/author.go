package store

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AuthorStore struct {
	db *sqlx.DB
}

func NewAuthorStore(db *sql.DB) *AuthorStore {
	return &AuthorStore{db: sqlx.NewDb(db, "postgres")}
}

type Author struct {
	ID    int       `db:"id" json:"-"`
	UID   uuid.UUID `db:"uid" json:"id,omitempty"`
	Name  string    `db:"name" json:"name"`
	Email string    `db:"email" json:"email"`
}
