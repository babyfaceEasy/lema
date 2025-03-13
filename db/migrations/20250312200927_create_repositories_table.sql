-- +goose Up
CREATE TABLE IF NOT EXISTS repositories (
    id bigserial NOT NULL PRIMARY KEY,
    uid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    name VARCHAR(200),
    description TEXT,
    url VARCHAR(200),
    programming_language VARCHAR(200),
    forks_count BIGINT,
    stars_count BIGINT,
    watchers_count BIGINT,
    open_issues_count BIGINT,
    since_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS repositories;
