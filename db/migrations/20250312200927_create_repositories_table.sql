-- +goose Up
CREATE TABLE IF NOT EXISTS repositories (
    id bigserial NOT NULL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    owner_name VARCHAR(200) NOT NULL,
    description TEXT,
    url VARCHAR(200) NOT NULL,
    programming_language VARCHAR(200),
    forks_count BIGINT,
    stars_count BIGINT,
    watchers_count BIGINT,
    open_issues_count BIGINT,
    until_date TIMESTAMPTZ,
    since_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, owner_name)
);

-- Create an index for quick lookup by name and owner
CREATE UNIQUE INDEX IF NOT EXISTS idx_repositories_name_owner ON repositories(name, owner_name);

-- +goose Down
DROP TABLE IF EXISTS repositories;
