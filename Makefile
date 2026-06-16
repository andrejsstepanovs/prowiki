.PHONY: build test lint migrate run serve

build:
	go build -o bin/prowiki ./cmd

test:
	go test -race ./...

lint:
	go vet ./...

migrate:
	@echo "Migrate command not implemented yet"

run:
	go run ./cmd

serve:
	go run ./cmd serve
