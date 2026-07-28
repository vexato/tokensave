.PHONY: build test lint install

build:
	go build -o bin/tokensave ./cmd/tokensave

test:
	go test ./...

lint:
	go vet ./...

install:
	go install ./cmd/tokensave
