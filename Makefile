.PHONY: build test lint migrate run serve

build:
	go build -o bin/prowiki ./cmd/prowiki

test:
	go test -race ./...

lint:
	go vet ./...
	golangci-lint run ./...

migrate:
	go run ./cmd/prowiki migrate

run:
	go run ./cmd/prowiki daemon

serve:
	go run ./cmd/prowiki server
