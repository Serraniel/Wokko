.PHONY: build test lint generate run

build:
	go build -o bin/wokko-api ./cmd/server

test:
	go test ./... -v -cover

lint:
	go vet ./...

generate:
	sqlc generate

run:
	go run ./cmd/server
