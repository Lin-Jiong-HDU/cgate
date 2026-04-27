.PHONY: build test lint run tidy clean

BINARY_NAME=app

build:
	go build -o bin/$(BINARY_NAME) .

test:
	go test ./... -v -cover

lint:
	golangci-lint run

run:
	go run .

tidy:
	go mod tidy

clean:
	rm -rf bin/
