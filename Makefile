.DEFAULT_GOAL := build

build:
	go build .

clean:
	rm -f lockoci

lint:
	golangci-lint run ./...

.PHONY: build clean lint
