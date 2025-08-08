BINARY=llm
SRC=main.go

.PHONY: all build clean

all: build

build:
	go build -o $(BINARY) $(SRC)

clean:
	rm -f $(BINARY)
