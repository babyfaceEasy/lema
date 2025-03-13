-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" schema pg_catalog;

-- +goose Down
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;