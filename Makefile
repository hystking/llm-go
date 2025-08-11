BINARY=llmx

# Version metadata (override VERSION to inject a specific tag)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
DATE    := $(shell date -u +%Y-%m-%d)

LDFLAGS := -s -w \
	-X llmx/pkg/version.Version=$(VERSION) \
	-X llmx/pkg/version.Commit=$(COMMIT) \
	-X llmx/pkg/version.Date=$(DATE)

.PHONY: all build clean install test release lint

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

clean:
	rm -f $(BINARY)

install: build
	mv $(BINARY) ~/bin/

test:
	go test -v ./...

lint:
	golangci-lint run

release:
	goreleaser release --clean
