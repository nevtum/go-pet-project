up:
	docker-compose up -d

down:
	docker-compose down

migrate:
	go run cmd/db_migrations/main.go

dev:
	go run cmd/server/main.go

projections:
	go run cmd/projections/main.go

test-unit:
	go test -v --race ./...

test-integration:
	go test -tags=integration -v --race ./...
