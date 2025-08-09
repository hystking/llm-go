BINARY=llm
SRC=main.go parser.go

.PHONY: all build clean install test

all: build

build:
	go build -o $(BINARY) $(SRC)

clean:
	rm -f $(BINARY)

install: build
	mv $(BINARY) /usr/local/bin/

test:
	go test -v ./...
