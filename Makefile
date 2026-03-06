.PHONY: build test run clean docker-build docker-up docker-down migrate lint

BINARY_NAME=lora-trainer
GO=go

build:
	$(GO) build -o bin/$(BINARY_NAME) ./cmd/server

test:
	$(GO) test ./... -v -race -count=1

test-short:
	$(GO) test ./... -v -short

run: build
	./bin/$(BINARY_NAME) -config configs/config.example.yaml

clean:
	rm -rf bin/
	$(GO) clean

lint:
	golangci-lint run ./...

# Docker
docker-build:
	docker build -t $(BINARY_NAME):latest -f docker/api/Dockerfile .

docker-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.yml down

docker-logs:
	docker compose -f deployments/docker-compose.yml logs -f

# Trainer images
trainer-images:
	./scripts/build-trainer-images.sh

# Database
migrate:
	psql "$${DATABASE_URL:-postgres://lora:lora@localhost:5432/lora_trainer?sslmode=disable}" \
		-f migrations/001_create_jobs.sql

# Dependencies
deps:
	$(GO) mod tidy
	$(GO) mod download

# All
all: deps lint test build
