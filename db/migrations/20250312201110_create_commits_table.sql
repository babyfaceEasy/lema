-- +goose Up
CREATE TABLE IF NOT EXISTS commits (
    id bigserial NOT NULL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    author_id BIGINT REFERENCES authors(id) ON DELETE CASCADE,
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    url VARCHAR(200),
    sha VARCHAR(200) NOT NULL,
    message TEXT,
    commit_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index on date for efficient sorting and filtering by date
CREATE INDEX IF NOT EXISTS idx_commits_date ON commits(commit_date);

-- +goose Down
DROP TABLE IF EXISTS commits;
