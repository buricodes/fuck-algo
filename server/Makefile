BIN := bin/server

.PHONY: run test lint build fmt

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)
	go vet ./...

fmt:
	gofmt -w .

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/server
