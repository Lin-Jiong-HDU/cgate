.PHONY: build test lint run tidy clean docker-build docker-build-runner docker-build-all docker-up docker-down

BINARY_NAME=cgate

build:
	go build -o bin/$(BINARY_NAME) ./cmd/

test:
	go test ./... -v -cover

lint:
	golangci-lint run

run:
	go run ./cmd/

tidy:
	go mod tidy

clean:
	rm -rf bin/

docker-build:
	docker build -t cgate:latest .

docker-build-runner:
	docker build -t claude-code-runner:latest ./runner-image/

docker-build-all: docker-build docker-build-runner

docker-up:
	docker compose up -d

docker-down:
	docker compose down
