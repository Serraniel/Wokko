SCRATCH ?= .e2e
GOBIN_TOOLS := $(shell go env GOPATH)/bin

.PHONY: dev test build generate types sqlc e2e docker

dev: ## Backend on :8080 with a local data folder. Run `cd frontend && npm run dev` beside it.
	cd backend && DATA_DIR=../$(SCRATCH)/data BACKUP_DIR=../$(SCRATCH)/backups go run ./cmd/wokko

test:
	cd backend && go vet ./... && go test ./...
	cd frontend && npm run check && npm test

build:
	cd frontend && npm run build
	cp -r frontend/build/. backend/internal/web/dist/
	cd backend && CGO_ENABLED=0 go build -o ../wokko ./cmd/wokko

generate: sqlc types

sqlc: ## Regenerates database code from backend/internal/store/queries.
	docker run --rm -u "$$(id -u):$$(id -g)" -v "$$PWD/backend":/src -w /src sqlc/sqlc:latest generate

types: ## Regenerates the frontend's API types from backend/internal/app/types.go.
	cd backend && go run github.com/gzuidhof/tygo@latest generate

e2e: build ## Runs the browser flows against a fresh instance.
	@mkdir -p $(SCRATCH) frontend/e2e/screenshots
	@DATA_DIR=$$(mktemp -d) BACKUP_DIR=$$(mktemp -d) PORT=8765 ./wokko & pid=$$!; sleep 2; \
	(cd frontend && npm run e2e); status=$$?; kill $$pid; exit $$status

docker:
	docker compose build
