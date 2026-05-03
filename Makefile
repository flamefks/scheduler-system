test:
	go test ./internal/...

test-integration:
	go test ./internal/... -tags=integration 

bench:
	go test ./internal/... -tags=integration -bench=. -benchmem -run=^$
