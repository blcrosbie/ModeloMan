.PHONY: run cli build test fmt vet

run:
	go run ./cmd/modeloman-server

cli:
	go run ./cmd/modeloman-cli --help

build:
	go build ./...

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...
