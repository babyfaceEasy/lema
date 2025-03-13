# Usage: make create-migrate name=create_users_table
create-migrate:
	goose -dir ./db/migrations create ${name} sql

mocks:
	mockgen -source internal/integrations/githubapi/client.go -destination internal/integrations/githubapi/mock_httpclient/client.go -package HttpClient