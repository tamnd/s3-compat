TARGET ?= minio

.PHONY: test test-json test-all matrix up down clean-buckets lint

test:
	S3COMPAT_TARGET=$(TARGET) go test ./... -v -timeout 300s

test-json:
	S3COMPAT_TARGET=$(TARGET) go test ./... -v -timeout 300s -json | tee results-$(TARGET).json

test-all:
	for t in minio seaweedfs garage rustfs liteio; do \
		S3COMPAT_TARGET=$$t go test ./... -v -timeout 300s -json | tee results-$$t.json; \
	done

matrix:
	go run ./scripts/gen-matrix results-*.json > COMPAT.md
	cat COMPAT.md

up:
	docker compose -f docker/docker-compose.yml up -d $(TARGET)

down:
	docker compose -f docker/docker-compose.yml down

clean-buckets:
	S3COMPAT_TARGET=$(TARGET) go run ./scripts/clean-buckets

lint:
	golangci-lint run ./...
