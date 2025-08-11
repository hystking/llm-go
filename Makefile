BINARY=llmx

.PHONY: all build clean install test release lint

all: build

build:
	go build -o $(BINARY) .

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
