BINARY=llmx

.PHONY: all build clean install test

all: build

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)

install: build
	mv $(BINARY) ~/bin/

test:
	go test -v ./...
