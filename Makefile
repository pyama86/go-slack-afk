.PHONY: lint fmt ci test test-redis test-dynamodb devdeps build run
LINTER := golangci-lint
build:
	go build --o bin/afk .
ci: devdeps lint test
run:
	go run .

lint:
	@echo ">> Running linter ($(LINTER))"
	$(LINTER) run

fmt:
	@echo ">> Formatting code"
	gofmt -w .
	goimports -w .

test:
	@echo ">> Running tests"
	@echo ">> Starting DynamoDB Local and Redis..."
	@if docker-compose up -d dynamodb-local redis 2>/dev/null; then \
		echo ">> Waiting for services to be ready..."; \
		sleep 3; \
		echo ">> Running all tests (including DynamoDB and Redis)..."; \
		DYNAMO_LOCAL=1 REDIS_URL=redis://localhost:6379 go test -v ./...; \
	else \
		echo ">> Warning: Could not start services (Docker may not be running)"; \
		echo ">> Running tests without external services (some tests may be skipped)..."; \
		go test -v ./...; \
	fi
	@echo ">> Tests completed"

test-redis:
	@echo ">> Running tests (Redis only)"
	@echo ">> Starting Redis..."
	@docker-compose up -d redis 2>/dev/null || true
	@sleep 2
	REDIS_URL=redis://localhost:6379 go test -v ./store/... -run TestRedisClient

test-dynamodb:
	@echo ">> Running DynamoDB tests"
	@docker-compose up -d dynamodb-local
	@echo ">> Waiting for DynamoDB Local to be ready..."
	@sleep 3
	DYNAMO_LOCAL=1 go test -v -cover ./store/... -run TestDynamoDBClient

devdeps:
	@echo ">> Installing development dependencies"
	which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
