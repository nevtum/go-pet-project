up:
	docker-compose up -d

down:
	docker-compose down

migrate:
	@go run cmd/db_migrations/main.go

dev:
	go run cmd/server/main.go
