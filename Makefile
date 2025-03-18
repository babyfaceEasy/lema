# Usage: make create-migrate name=create_users_table
create-migrate:
	goose -dir ./db/migrations create ${name} sql

# Run the migrations
migrate-up:
	@echo "Running migrations..."
	goose -dir ./db/migrations postgres "$$DATABASE_URL" up

# Run the migrations
migrate-down:
	@echo "Running migrations..."
	goose -dir ./db/migrations postgres "$$DATABASE_URL" down

# Seed the DB with fixed data
seed:
	@echo "Running seeders..."
	go run cmd/cli/seed.go

# Seed the DB with live data
seed-live:
	@echo "Running seeders..."
	chmod +x seed_db.sh
	./seed_db.sh

# Start the app using go run
run:
	@echo "Starting the app locally using go run..."
	go run ./cmd/apiserver/main.go


build:
	@echo "Building the application..."
	go build -o bin/main ./cmd/apiserver/main.go

mocks:
	mockgen -source internal/integrations/githubapi/client.go -destination internal/integrations/githubapi/mock_httpclient/client.go -package HttpClient

# run tests
test:
	@echo "Starting the app locally using go run..."
	go test ./...