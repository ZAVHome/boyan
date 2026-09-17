.PHONY: all build clean test lint docker-up docker-down swagger run-backend run-desktop run-mobile

# Variables
BINARY_NAME=opds-server
BUILD_DIR=bin
GO_FILES=$(shell find backend -name '*.go' 2>/dev/null)

all: build

## -------------------------------------------------------------
## Build Targets
## -------------------------------------------------------------
build: build-backend build-frontends

build-backend:
	@echo "==> Building Go backend binary..."
	@mkdir -p $(BUILD_DIR)
	cd backend && go build -ldflags="-s -w" -o ../$(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

build-linux:
	@echo "==> Building Go backend binary for Linux (WSL2)..."
	@mkdir -p $(BUILD_DIR)
	cd backend && set GOOS=linux&& set GOARCH=amd64&& set CGO_ENABLED=0&& go build -ldflags="-s -w" -o ../$(BUILD_DIR)/boyan-linux ./cmd/server

deploy-wsl: build-linux
	@echo "==> Deploying to WSL2 Ubuntu..."
	wsl -d Ubuntu -u root -- bash scripts/wsl/deploy.sh

build-frontends: build-desktop build-mobile

build-desktop:
	@echo "==> Building Desktop Web Frontend..."
	cd frontends/web-desktop && npm install && npm run build

build-mobile:
	@echo "==> Building Mobile PWA Frontend..."
	cd frontends/web-mobile && npm install && npm run build

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf frontends/web-desktop/dist
	rm -rf frontends/web-mobile/dist

## -------------------------------------------------------------
## Development Commands
## -------------------------------------------------------------
run-backend:
	@echo "==> Starting backend dev server..."
	cd backend && go run ./cmd/server/main.go --config ../config.yaml

run-desktop:
	@echo "==> Starting desktop frontend Vite dev server..."
	cd frontends/web-desktop && npm run dev

run-mobile:
	@echo "==> Starting mobile frontend Vite dev server..."
	cd frontends/web-mobile && npm run dev

## -------------------------------------------------------------
## Testing & Quality
## -------------------------------------------------------------
test: test-backend test-frontends

test-backend:
	@echo "==> Running Go tests..."
	cd backend && go test -v -race ./...

test-frontends:
	@echo "==> Running frontend tests..."
	cd frontends/web-desktop && npm test || true
	cd frontends/web-mobile && npm test || true

lint:
	@echo "==> Running linters..."
	cd backend && golangci-lint run || true
	cd frontends/web-desktop && npm run lint || true
	cd frontends/web-mobile && npm run lint || true

## -------------------------------------------------------------
## API Docs & Generators
## -------------------------------------------------------------
swagger:
	@echo "==> Generating OpenAPI / Swagger specifications..."
	cd backend && swag init -g cmd/server/main.go -o docs/swagger

## -------------------------------------------------------------
## Docker
## -------------------------------------------------------------
docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
