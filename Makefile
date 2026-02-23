.PHONY: run cli build test fmt vet mm modeloman install-modeloman

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

mm:
	go run ./cmd/mm --help

modeloman:
	go run ./cmd/modeloman --help

install-modeloman:
	go install ./cmd/modeloman
