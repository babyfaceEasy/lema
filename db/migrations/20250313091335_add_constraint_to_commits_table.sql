-- +goose Up
ALTER TABLE commits ADD CONSTRAINT unique_repo_commit UNIQUE (repository_id, sha);


-- +goose Down
ALTER TABLE commits DROP CONSTRAINT unique_repo_commit;
