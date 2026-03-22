APP_NAME := zhajinhua
BACKEND_DIR := backend
FRONTEND_DIR := frontend
FRONTEND_TG_DIR := frontend-tg

.PHONY: build dev test clean docker-build docker-run docker-up build-frontend

## build: Build the Go backend binary (CGO required for SQLite)
build:
	cd $(BACKEND_DIR) && CGO_ENABLED=1 go build -o $(APP_NAME) .

## dev: Run the backend locally in dev mode
dev:
	cd $(BACKEND_DIR) && CGO_ENABLED=1 go run .

## test: Run all Go tests
test:
	cd $(BACKEND_DIR) && CGO_ENABLED=1 go test ./...

## build-frontend: Build all frontend apps (te, pe, tg)
build-frontend:
	cd $(FRONTEND_DIR) && npm install && npm run build:te && npm run build:pe
	cd $(FRONTEND_TG_DIR) && npm install && npm run build

## docker-build: Build the Docker image
docker-build:
	docker build -t $(APP_NAME) .

## docker-run: Run the Docker container
docker-run:
	docker run --rm -it \
		-p 8080:8080 \
		-e APP_MODE=$${APP_MODE:-te} \
		-e ADMIN_PASSWORD=$${ADMIN_PASSWORD:-admin123} \
		-v zhajinhua-data:/app/data \
		$(APP_NAME)

## docker-up: Start with docker compose
docker-up:
	docker compose up -d --build

## docker-down: Stop docker compose services
docker-down:
	docker compose down

## clean: Remove build artifacts and databases
clean:
	rm -f $(BACKEND_DIR)/$(APP_NAME)
	rm -f $(BACKEND_DIR)/*.db $(BACKEND_DIR)/*.db-shm $(BACKEND_DIR)/*.db-wal

## help: Show available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
