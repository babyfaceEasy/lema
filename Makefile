# Usage: make create-migrate name=create_users_table
create-migrate:
	goose -dir ./db/migrations create ${name} sql