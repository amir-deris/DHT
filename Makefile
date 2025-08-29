.PHONY: build run test lint

build:
	go build ./...

run:
	go run ./cmd/dhtnode --bind=:8080

test:
	go test ./...


