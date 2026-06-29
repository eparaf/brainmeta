VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X disci/brain/internal/api.Version=$(VERSION)

.PHONY: build build-prod run sim chat test vet lint tidy docker clean

build:        ## dependency-free dev build
	go build ./...

build-prod:   ## production binary with Postgres + Redis
	CGO_ENABLED=0 go build -tags "pgx redis" -ldflags="$(LDFLAGS)" -o bin/brain ./cmd/brain

run:          ## run the HTTP server
	go run ./cmd/brain serve

sim:          ## run the 30-day simulation
	go run ./cmd/brain sim

chat:         ## run the mock-agent chat demo
	go run ./cmd/brain chat

test:         ## run all tests with the race detector
	go test -race -timeout 120s ./...

vet:
	go vet ./...

lint:         ## requires golangci-lint
	golangci-lint run ./...

tidy:
	go mod tidy

docker:
	docker build -t disci/brain:$(VERSION) .

up:           ## full local stack (brain + postgres + redis)
	docker compose up --build

down:
	docker compose down

clean:
	rm -rf bin brain-data
