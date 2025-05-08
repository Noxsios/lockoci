.DEFAULT_GOAL := build

build:
	go build .

test:
	go test ./... -v

clean:
	rm -f lockoci

lint:
	golangci-lint run ./...

.PHONY: build test clean lint
