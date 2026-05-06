SQLC_VERSION := 1.27.0
MIGRATIONS_DIR := sql/migrations

test:
	go test ./internal/...

test-integration:
	go test ./internal/... -tags=integration 

bench:
	go test ./internal/... -tags=integration -bench=. -benchmem -run=^$

sqlc:
	docker run --rm \
		-v "$$PWD:/src" \
		-w /src \
		sqlc/sqlc:$(SQLC_VERSION) \
		generate

migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
	