package domain

import (
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	ID                  int        `db:"id" json:"-"`
	UID                 uuid.UUID  `db:"uid" json:"id,omitempty"`
	Name                string     `db:"name" json:"name"`
	OwnerName           string     `db:"owner_name" json:"owner_name"` // todo: change this to owner later
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

type Commit struct {
	ID           int        `db:"id" json:"-"`
	UID          uuid.UUID  `db:"uid" json:"id,omitempty"`
	RepositoryID int        `db:"repository_id" json:"-"`
	AuthorID     int        `db:"author_id" json:"-"`
	SHA          string     `db:"sha" json:"sha"`
	URL          string     `db:"url" json:"url"`
	Message      string     `db:"message" json:"message"`
	Date         time.Time  `db:"date" json:"date"` // todo: change this to commit_date
	CreatedAt    time.Time  `db:"created_at" json:"-"`
	Repository   Repository `db:"Repository" json:"repository"`
	Author       Author     `db:"Author" json:"author"`
}

type Author struct {
	ID    int       `db:"id" json:"-"`
	UID   uuid.UUID `db:"uid" json:"id,omitempty"`
	Name  string    `db:"name" json:"name"`
	Email string    `db:"email" json:"email"`
}

type CommitAuthor struct {
	Author
	CommitCount int `db:"commit_count" json:"commit_count"`
}

type PaginatedCommits struct{}

type Pagination struct{}
