.PHONY: build test lint install benchmark

build:
	go build -o bin/tokensave ./cmd/tokensave

test:
	go test ./...

lint:
	go vet ./...

install:
	go install ./cmd/tokensave

benchmark:
	sh scripts/benchmark.sh
