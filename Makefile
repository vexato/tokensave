.PHONY: build test lint install benchmark demo demo-check

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

demo-check:
	@command -v go >/dev/null || { echo "error: go is required" >&2; exit 1; }
	@command -v vhs >/dev/null || { echo "error: vhs is required (https://github.com/charmbracelet/vhs#installation)" >&2; exit 1; }
	@command -v ttyd >/dev/null || { echo "error: ttyd is required by vhs" >&2; exit 1; }
	@command -v ffmpeg >/dev/null || { echo "error: ffmpeg is required" >&2; exit 1; }

demo: demo-check
	$(MAKE) build
	vhs docs/demo.tape
