# Wokko Full Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build Wokko — a self-hosted recipe management web app with Go API backend, SvelteKit frontend, PostgreSQL database, Docker deployment, and full CI/CD.

**Architecture:** Go (Echo) REST API at `/api/` + SvelteKit frontend. Separated services in Docker Compose. PostgreSQL for data, local filesystem volume for photo uploads. kitshn-compatible-compatible API layer for kitshn app support.

**Tech Stack:** Go 1.22, Echo v4, sqlc, golang-migrate, pgx v5, go-webauthn, golang-jwt/v5 | SvelteKit, TailwindCSS v4, Paraglide-JS, shadcn-svelte | PostgreSQL 16 | Docker + Docker Compose | GitHub Actions

**Important:** All git commits must use `git -c commit.gpgsign=false commit`. Never use `--gpg-sign` or default signing.

---

## Task 1: Repository Foundation

**Files:**
- Create: `LICENSE`
- Create: `README.md`
- Create: `CLAUDE.md`
- Create: `.gitignore`
- Create: `.editorconfig`

**Step 1: Create LICENSE (EUPL-1.2)**

Download and write EUPL-1.2 license text to `LICENSE`. Full text from: https://joinup.ec.europa.eu/sites/default/files/custom-page/attachment/2020-03/EUPL-1.2%20EN.txt

**Step 2: Create .gitignore**

```
# Go
*.exe
*.test
*.prof
/backend/vendor/
/backend/tmp/

# Node
node_modules/
.svelte-kit/
build/
dist/
.env
.env.local
.env.*.local

# Docker
.env.docker

# OS
.DS_Store
Thumbs.db

# IDE
.idea/
.vscode/
*.swp

# Uploads (local dev)
/backend/uploads/

# Tmp
/tmp/
```

**Step 3: Create README.md**

Include: project name + logo placeholder, description, features list, quick start with Docker Compose, environment variables table, backup/restore instructions, kitshn compatibility note, contributing section, license badge.

**Step 4: Create CLAUDE.md**

```markdown
# Wokko — Claude Development Guidelines

## Project Overview
Wokko is a self-hosted recipe management app. Go backend API + SvelteKit frontend + PostgreSQL.

## Repository Structure
- `backend/` — Go REST API (Echo framework)
- `frontend/` — SvelteKit web application
- `docs/` — Design docs and plans
- `.github/` — CI/CD workflows and Dependabot config

## Commit Convention
Use Conventional Commits with these scopes:
- `feat(auth)` — Authentication, user management, passkeys, invites
- `feat(recipes)` — Recipe CRUD, steps, photos
- `feat(ingredients)` — Ingredient management and suggestions
- `feat(units)` — Unit management
- `feat(sharing)` — Public recipes, share links
- `feat(mcp)` — Model Context Protocol integration
- `feat(i18n)` — Internationalization (EN/DE)
- `feat(ui)` — Frontend UI components
- `fix(*)` — Bug fixes (use relevant scope)
- `chore(infra)` — Docker, CI, dependencies
- `chore(ci)` — GitHub Actions workflows

## ALWAYS
- Use `git -c commit.gpgsign=false commit` for all commits (no GPG signing)
- Run tests before committing: `cd backend && go test ./...` and `cd frontend && npm test`
- Keep backend and frontend Dockerfiles using multi-stage builds
- Use sqlc for all database queries — never raw SQL in application code
- All API endpoints must have corresponding tests in `backend/internal/api/*_test.go`

## NEVER
- Add features beyond what is requested
- Skip database migrations — all schema changes via `backend/migrations/`
- Store secrets in code — use environment variables
- Use SQLite — PostgreSQL only
- Skip error handling in Go — always handle errors explicitly

## Changelog Categories (for GitHub PR labels and release notes)
Auth, Recipes, Ingredients, Units, Sharing, MCP, i18n, UI, Infra, Fix

## Architecture Decisions
- JWT access tokens (15min) + refresh tokens (httpOnly cookie, 7 days)
- Passkeys via WebAuthn (go-webauthn library)
- Photos stored on filesystem at UPLOAD_PATH env var (default: /data/uploads)
- kitshn-compatible v1 API compatibility at /api/ for kitshn mobile app support
- Semantic versioning starting at 0.1.0
- Footer version auto-populated from VITE_APP_VERSION env var (set at build time from git tag)
```

**Step 5: Create .editorconfig**

```ini
root = true

[*]
indent_style = space
indent_size = 2
end_of_line = lf
charset = utf-8
trim_trailing_whitespace = true
insert_final_newline = true

[*.go]
indent_style = tab
indent_size = 4

[Makefile]
indent_style = tab
```

**Step 6: Create top-level Makefile**

```makefile
.PHONY: dev dev-down build test lint

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

dev-down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

build:
	$(MAKE) -C backend build
	$(MAKE) -C frontend build

test:
	$(MAKE) -C backend test
	$(MAKE) -C frontend test

lint:
	$(MAKE) -C backend lint
	$(MAKE) -C frontend lint
```

**Step 7: Commit**

```bash
git -c commit.gpgsign=false commit -m "chore(infra): initialize repository with license, readme, and dev config"
```

---

## Task 2: GitHub Repository Setup

**Requires:** `gh` CLI authenticated

**Step 1: Create GitHub repository**

```bash
gh repo create Serraniel/Wokko --public --description "Self-hosted recipe management — fast, lightweight kitshn-compatible alternative" --homepage "" --push --source .
```

**Step 2: Create GitHub labels for PR categories**

```bash
gh label create "Auth" --color "0075ca" --description "Authentication, users, passkeys"
gh label create "Recipes" --color "e4e669" --description "Recipe management"
gh label create "Ingredients" --color "a2eeef" --description "Ingredient management"
gh label create "Units" --color "d93f0b" --description "Unit management"
gh label create "Sharing" --color "0e8a16" --description "Sharing, public access"
gh label create "MCP" --color "5319e7" --description "Model Context Protocol"
gh label create "i18n" --color "f9d0c4" --description "Internationalization"
gh label create "UI" --color "fef2c0" --description "Frontend UI"
gh label create "Infra" --color "bfd4f2" --description "Infrastructure, Docker, CI"
gh label create "Fix" --color "d73a4a" --description "Bug fix"
```

**Step 3: Create milestones**

```bash
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v0.1.0 — Foundation" -f description="Repo setup, CI/CD, auth, basic recipe CRUD"
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v0.2.0 — Recipe Features" -f description="Steps, ingredients, units, photos, tags"
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v0.3.0 — Sharing" -f description="Public recipes, share links, public gallery"
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v0.4.0 — kitshn Compat" -f description="kitshn-compatible API compatibility layer for kitshn app"
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v0.5.0 — Admin & UX" -f description="Admin panel, user management, invite links"
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v0.6.0 — MCP" -f description="Model Context Protocol, OAuth"
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v0.7.0 — i18n" -f description="Full EN/DE, seed data localization"
gh api repos/Serraniel/Wokko/milestones -X POST -f title="v1.0.0 — Stable" -f description="All features, docs, performance audit"
```

**Step 4: Create GitHub issues**

Create issues for major feature areas (one per bullet, assign to correct milestone):

Milestone 1 (v0.1.0):
- "Initialize repository structure and CI/CD pipeline" [Infra]
- "Implement user authentication with JWT and passkeys" [Auth]
- "First-run setup wizard (admin account creation)" [Auth]
- "Basic recipe CRUD API" [Recipes]
- "SvelteKit frontend scaffold with routing" [UI]

Milestone 2 (v0.2.0):
- "Recipe step editor with per-step ingredients" [Recipes]
- "Photo upload for recipes" [Recipes]
- "Ingredient management with autocomplete" [Ingredients]
- "Unit management with autocomplete" [Units]
- "Tags system for recipes" [Recipes]
- "Default seed data (ingredients, units, tags)" [Ingredients]

Milestone 3 (v0.3.0):
- "Public recipe gallery (no login required)" [Sharing]
- "Per-recipe share links" [Sharing]
- "All-recipes share link for users" [Sharing]

Milestone 4 (v0.4.0):
- "kitshn-compatible v1 API compatibility layer for kitshn" [Recipes]

Milestone 5 (v0.5.0):
- "Admin panel: user management" [Auth]
- "Invite link generation and management" [Auth]
- "Open/closed registration settings" [Auth]
- "Ingredient and unit merge/alias management" [Ingredients]

Milestone 6 (v0.6.0):
- "MCP protocol endpoints (opt-in)" [MCP]
- "OAuth 2.0 for MCP authentication" [MCP]

Milestone 7 (v0.7.0):
- "German (DE) translation of all UI strings" [i18n]
- "Localized seed data (EN + DE)" [i18n]

Use `gh issue create` for each.

---

## Task 3: Backend Scaffold

**Files:**
- Create: `backend/go.mod`
- Create: `backend/go.sum`
- Create: `backend/cmd/server/main.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/Dockerfile`
- Create: `backend/Makefile`

**Step 1: Initialize Go module**

```bash
cd backend
go mod init github.com/serraniel/wokko/backend
```

**Step 2: Add core dependencies**

```bash
go get github.com/labstack/echo/v4@latest
go get github.com/labstack/echo-jwt/v4@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/go-webauthn/webauthn@latest
go get golang.org/x/crypto@latest
go get github.com/google/uuid@latest
go get github.com/joho/godotenv@latest
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/golang-migrate/migrate/v4/database/postgres@latest
go get github.com/golang-migrate/migrate/v4/source/file@latest
```

**Step 3: Create config package**

`backend/internal/config/config.go`:
```go
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL       string
	JWTSecret         string
	UploadPath        string
	PublicURL         string
	Port              string
	OpenRegistration  bool
	AllowPublicRecipes bool
	MCPEnabled        bool
	RPDisplayName     string // WebAuthn relying party display name
	RPID              string // WebAuthn relying party ID (domain)
	RPOrigin          string // WebAuthn origin
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:3000"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	uploadPath := os.Getenv("UPLOAD_PATH")
	if uploadPath == "" {
		uploadPath = "/data/uploads"
	}
	openReg, _ := strconv.ParseBool(os.Getenv("OPEN_REGISTRATION"))
	allowPublic, _ := strconv.ParseBool(os.Getenv("ALLOW_PUBLIC_RECIPES"))
	if os.Getenv("ALLOW_PUBLIC_RECIPES") == "" {
		allowPublic = true
	}
	mcpEnabled, _ := strconv.ParseBool(os.Getenv("MCP_ENABLED"))

	rpDisplayName := os.Getenv("RP_DISPLAY_NAME")
	if rpDisplayName == "" {
		rpDisplayName = "Wokko"
	}
	rpID := os.Getenv("RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}
	rpOrigin := os.Getenv("RP_ORIGIN")
	if rpOrigin == "" {
		rpOrigin = "http://localhost:3000"
	}

	return &Config{
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		UploadPath:         uploadPath,
		PublicURL:          publicURL,
		Port:               port,
		OpenRegistration:   openReg,
		AllowPublicRecipes: allowPublic,
		MCPEnabled:         mcpEnabled,
		RPDisplayName:      rpDisplayName,
		RPID:               rpID,
		RPOrigin:           rpOrigin,
	}, nil
}
```

**Step 4: Create main.go**

`backend/cmd/server/main.go`:
```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/serraniel/wokko/backend/internal/config"
	"github.com/serraniel/wokko/backend/internal/db"
	"github.com/serraniel/wokko/backend/internal/api"
)

func main() {
	// Load .env in development
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// Run migrations
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	// Connect to DB
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	defer pool.Close()

	// Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Register routes
	api.RegisterRoutes(e, pool, cfg)

	// Graceful shutdown
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
```

**Step 5: Create backend Dockerfile**

`backend/Dockerfile`:
```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /wokko-api ./cmd/server

# Final stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /wokko-api /wokko-api
COPY --from=builder /app/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/wokko-api"]
```

**Step 6: Create backend Makefile**

```makefile
.PHONY: build test lint generate run

build:
	go build -o bin/wokko-api ./cmd/server

test:
	go test ./... -v -cover

lint:
	golangci-lint run

generate:
	sqlc generate

run:
	go run ./cmd/server
```

**Step 7: Compile check**

```bash
cd backend && go build ./...
```

Expected: no errors (some packages not yet created will need stub files).

**Step 8: Commit**

```bash
git -c commit.gpgsign=false commit -m "chore(infra): scaffold Go backend with Echo framework"
```

---

## Task 4: Database Migrations

**Files:**
- Create: `backend/internal/db/connect.go`
- Create: `backend/internal/db/migrate.go`
- Create: `backend/migrations/000001_init.up.sql`
- Create: `backend/migrations/000001_init.down.sql`
- Create: `backend/migrations/000002_seed_data.up.sql`
- Create: `backend/migrations/000002_seed_data.down.sql`
- Create: `backend/sqlc.yaml`

**Step 1: Create db/connect.go**

```go
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}
```

**Step 2: Create db/migrate.go**

```go
package db

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) error {
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Println("database migrations applied")
	return nil
}
```

**Step 3: Create 000001_init.up.sql**

```sql
-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Users
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username    VARCHAR(64) NOT NULL UNIQUE,
    email       VARCHAR(255) UNIQUE,
    display_name VARCHAR(128),
    password_hash VARCHAR(255),
    role        VARCHAR(16) NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Passkey credentials
CREATE TABLE passkey_credentials (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id   BYTEA NOT NULL UNIQUE,
    public_key      BYTEA NOT NULL,
    sign_count      BIGINT NOT NULL DEFAULT 0,
    name            VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Refresh tokens
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Invite links
CREATE TABLE invite_links (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token       VARCHAR(64) NOT NULL UNIQUE,
    created_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    used_by     UUID REFERENCES users(id),
    max_uses    INT NOT NULL DEFAULT 1,
    use_count   INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Setup state
CREATE TABLE system_settings (
    key     VARCHAR(128) PRIMARY KEY,
    value   TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO system_settings (key, value) VALUES ('setup_complete', 'false');

-- Tags
CREATE TABLE tags (
    id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name    VARCHAR(64) NOT NULL UNIQUE
);

-- Units
CREATE TABLE units (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_en         VARCHAR(128) NOT NULL,
    name_de         VARCHAR(128),
    abbreviation_en VARCHAR(32),
    abbreviation_de VARCHAR(32),
    plural_en       VARCHAR(128),
    plural_de       VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unit aliases
CREATE TABLE unit_aliases (
    id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    alias   VARCHAR(128) NOT NULL,
    UNIQUE(unit_id, alias)
);

-- Ingredients
CREATE TABLE ingredients (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_en         VARCHAR(256) NOT NULL,
    name_de         VARCHAR(256),
    plural_en       VARCHAR(256),
    plural_de       VARCHAR(256),
    default_unit_id UUID REFERENCES units(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ingredient aliases
CREATE TABLE ingredient_aliases (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ingredient_id   UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
    alias           VARCHAR(256) NOT NULL,
    UNIQUE(ingredient_id, alias)
);

-- Ingredient-unit usage stats (for smart default suggestions)
CREATE TABLE ingredient_unit_stats (
    ingredient_id   UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
    unit_id         UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    use_count       INT NOT NULL DEFAULT 1,
    PRIMARY KEY (ingredient_id, unit_id)
);

-- Recipes
CREATE TABLE recipes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(256) NOT NULL,
    description     TEXT,
    photo_path      VARCHAR(512),
    work_time_min   INT,
    servings        INT,
    is_public       BOOLEAN NOT NULL DEFAULT false,
    source_url      VARCHAR(1024),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Recipe tags (junction)
CREATE TABLE recipe_tags (
    recipe_id   UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (recipe_id, tag_id)
);

-- Recipe steps
CREATE TABLE recipe_steps (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id       UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_order      INT NOT NULL,
    description     TEXT NOT NULL,
    duration_min    INT,
    UNIQUE(recipe_id, step_order)
);

-- Step ingredients
CREATE TABLE step_ingredients (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    step_id         UUID NOT NULL REFERENCES recipe_steps(id) ON DELETE CASCADE,
    ingredient_id   UUID NOT NULL REFERENCES ingredients(id),
    unit_id         UUID REFERENCES units(id),
    amount          NUMERIC(10,3),
    notes           TEXT
);

-- Share links
CREATE TABLE share_links (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token       VARCHAR(64) NOT NULL UNIQUE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipe_id   UUID REFERENCES recipes(id) ON DELETE CASCADE, -- NULL = all user's recipes
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- MCP tokens (for MCP OAuth)
CREATE TABLE mcp_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(128),
    scopes      TEXT[] NOT NULL DEFAULT '{}',
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_recipes_author ON recipes(author_id);
CREATE INDEX idx_recipes_public ON recipes(is_public);
CREATE INDEX idx_recipe_steps_recipe ON recipe_steps(recipe_id);
CREATE INDEX idx_step_ingredients_step ON step_ingredients(step_id);
CREATE INDEX idx_ingredients_name_en ON ingredients USING gin(name_en gin_trgm_ops);
CREATE INDEX idx_ingredients_name_de ON ingredients USING gin(name_de gin_trgm_ops);
CREATE INDEX idx_units_name_en ON units USING gin(name_en gin_trgm_ops);
```

**Step 4: Create 000001_init.down.sql**

```sql
DROP TABLE IF EXISTS mcp_tokens;
DROP TABLE IF EXISTS share_links;
DROP TABLE IF EXISTS step_ingredients;
DROP TABLE IF EXISTS recipe_steps;
DROP TABLE IF EXISTS recipe_tags;
DROP TABLE IF EXISTS recipes;
DROP TABLE IF EXISTS ingredient_unit_stats;
DROP TABLE IF EXISTS ingredient_aliases;
DROP TABLE IF EXISTS ingredients;
DROP TABLE IF EXISTS unit_aliases;
DROP TABLE IF EXISTS units;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS system_settings;
DROP TABLE IF EXISTS invite_links;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS passkey_credentials;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "uuid-ossp";
```

**Step 5: Create seed data migration (000002_seed_data.up.sql)**

This file inserts ~30 common units and ~200 common ingredients in EN+DE. See full seed data list in `backend/migrations/seeds/` directory. Key units:
- Metric: gram (g), kilogram (kg), milligram (mg), milliliter (ml), liter (l), centiliter (cl)
- Volume: teaspoon (tsp), tablespoon (tbsp), cup, fluid ounce (fl oz)
- Count: piece (pcs), pinch, bunch, clove, can, package, slice, drop
- Cooking: handful, dash

For each unit insert with EN + DE names, abbreviations, plural forms.

For ingredients: common food items grouped by category (vegetables, fruits, proteins, dairy, grains, spices, liquids, baking). Each with EN + DE name and suggested default unit.

After inserts, mark setup as seeded:
```sql
INSERT INTO system_settings (key, value) VALUES ('data_seeded', 'true') ON CONFLICT DO NOTHING;
```

**Step 6: Create sqlc.yaml**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/db/queries"
    schema: "migrations"
    gen:
      go:
        package: "queries"
        out: "internal/db/queries"
        emit_json_tags: true
        emit_pointers_for_null_fields: true
        emit_prepared_queries: false
```

**Step 7: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(infra): add database schema migrations and seed data"
```

---

## Task 5: sqlc Queries

**Files:**
- Create: `backend/internal/db/queries/users.sql`
- Create: `backend/internal/db/queries/recipes.sql`
- Create: `backend/internal/db/queries/ingredients.sql`
- Create: `backend/internal/db/queries/units.sql`
- Create: `backend/internal/db/queries/auth.sql`

**Step 1: users.sql**

```sql
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (username, email, display_name, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET
  display_name = COALESCE($2, display_name),
  email = COALESCE($3, email),
  password_hash = COALESCE($4, password_hash),
  updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetSetupComplete :one
SELECT value FROM system_settings WHERE key = 'setup_complete';

-- name: SetSetupComplete :exec
INSERT INTO system_settings (key, value) VALUES ('setup_complete', 'true')
ON CONFLICT (key) DO UPDATE SET value = 'true', updated_at = NOW();
```

**Step 2: auth.sql**

```sql
-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3) RETURNING *;

-- name: GetRefreshToken :one
SELECT rt.*, u.id as uid, u.username, u.role FROM refresh_tokens rt
JOIN users u ON u.id = rt.user_id
WHERE rt.token_hash = $1 AND rt.expires_at > NOW();

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE token_hash = $1;

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens WHERE user_id = $1;

-- name: CreateInviteLink :one
INSERT INTO invite_links (token, created_by, max_uses, expires_at)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetInviteLink :one
SELECT * FROM invite_links WHERE token = $1;

-- name: UseInviteLink :exec
UPDATE invite_links SET use_count = use_count + 1, used_by = $2
WHERE token = $1;

-- name: ListInviteLinks :many
SELECT il.*, u.username as creator_username FROM invite_links il
JOIN users u ON u.id = il.created_by
ORDER BY il.created_at DESC;

-- name: CreatePasskeyCredential :one
INSERT INTO passkey_credentials (user_id, credential_id, public_key, sign_count, name)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetPasskeyCredential :one
SELECT pc.*, u.id as uid, u.username, u.role FROM passkey_credentials pc
JOIN users u ON u.id = pc.user_id
WHERE pc.credential_id = $1;

-- name: UpdatePasskeySignCount :exec
UPDATE passkey_credentials SET sign_count = $2 WHERE credential_id = $1;

-- name: ListUserPasskeys :many
SELECT * FROM passkey_credentials WHERE user_id = $1;

-- name: DeletePasskeyCredential :exec
DELETE FROM passkey_credentials WHERE id = $1 AND user_id = $2;
```

**Step 3: recipes.sql**

```sql
-- name: CreateRecipe :one
INSERT INTO recipes (author_id, name, description, photo_path, work_time_min, servings, is_public, source_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- name: GetRecipeByID :one
SELECT * FROM recipes WHERE id = $1;

-- name: ListRecipesByAuthor :many
SELECT * FROM recipes WHERE author_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListPublicRecipes :many
SELECT r.*, u.username as author_username, u.display_name as author_display_name
FROM recipes r JOIN users u ON u.id = r.author_id
WHERE r.is_public = true ORDER BY r.created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateRecipe :one
UPDATE recipes SET
  name = COALESCE($2, name),
  description = COALESCE($3, description),
  photo_path = COALESCE($4, photo_path),
  work_time_min = COALESCE($5, work_time_min),
  servings = COALESCE($6, servings),
  is_public = COALESCE($7, is_public),
  source_url = COALESCE($8, source_url),
  updated_at = NOW()
WHERE id = $1 RETURNING *;

-- name: DeleteRecipe :exec
DELETE FROM recipes WHERE id = $1;

-- name: CreateRecipeStep :one
INSERT INTO recipe_steps (recipe_id, step_order, description, duration_min)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListRecipeSteps :many
SELECT * FROM recipe_steps WHERE recipe_id = $1 ORDER BY step_order;

-- name: UpdateRecipeStep :one
UPDATE recipe_steps SET description = $2, duration_min = $3, step_order = $4
WHERE id = $1 RETURNING *;

-- name: DeleteRecipeStep :exec
DELETE FROM recipe_steps WHERE id = $1;

-- name: CreateStepIngredient :one
INSERT INTO step_ingredients (step_id, ingredient_id, unit_id, amount, notes)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListStepIngredients :many
SELECT si.*, i.name_en as ingredient_name, u.name_en as unit_name, u.abbreviation_en as unit_abbr
FROM step_ingredients si
JOIN ingredients i ON i.id = si.ingredient_id
LEFT JOIN units u ON u.id = si.unit_id
WHERE si.step_id = $1;

-- name: DeleteStepIngredients :exec
DELETE FROM step_ingredients WHERE step_id = $1;

-- name: AddRecipeTag :exec
INSERT INTO recipe_tags (recipe_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: RemoveRecipeTag :exec
DELETE FROM recipe_tags WHERE recipe_id = $1 AND tag_id = $2;

-- name: ListRecipeTags :many
SELECT t.* FROM tags t JOIN recipe_tags rt ON rt.tag_id = t.id WHERE rt.recipe_id = $1;
```

**Step 4: ingredients.sql**

```sql
-- name: SearchIngredients :many
SELECT * FROM ingredients
WHERE name_en ILIKE '%' || $1 || '%' OR name_de ILIKE '%' || $1 || '%'
ORDER BY name_en LIMIT $2;

-- name: GetIngredientByID :one
SELECT * FROM ingredients WHERE id = $1;

-- name: CreateIngredient :one
INSERT INTO ingredients (name_en, name_de, plural_en, plural_de, default_unit_id)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateIngredient :one
UPDATE ingredients SET
  name_en = COALESCE($2, name_en),
  name_de = COALESCE($3, name_de),
  default_unit_id = COALESCE($4, default_unit_id)
WHERE id = $1 RETURNING *;

-- name: MergeIngredient :exec
UPDATE step_ingredients SET ingredient_id = $2 WHERE ingredient_id = $1;

-- name: DeleteIngredient :exec
DELETE FROM ingredients WHERE id = $1;

-- name: GetSuggestedUnitForIngredient :one
SELECT unit_id, use_count FROM ingredient_unit_stats
WHERE ingredient_id = $1 ORDER BY use_count DESC LIMIT 1;

-- name: IncrementIngredientUnitStat :exec
INSERT INTO ingredient_unit_stats (ingredient_id, unit_id, use_count)
VALUES ($1, $2, 1)
ON CONFLICT (ingredient_id, unit_id) DO UPDATE SET use_count = ingredient_unit_stats.use_count + 1;

-- name: ListIngredients :many
SELECT * FROM ingredients ORDER BY name_en LIMIT $1 OFFSET $2;

-- name: CountIngredients :one
SELECT COUNT(*) FROM ingredients;
```

**Step 5: units.sql**

```sql
-- name: SearchUnits :many
SELECT * FROM units
WHERE name_en ILIKE '%' || $1 || '%' OR name_de ILIKE '%' || $1 || '%'
   OR abbreviation_en ILIKE '%' || $1 || '%'
ORDER BY name_en LIMIT $2;

-- name: GetUnitByID :one
SELECT * FROM units WHERE id = $1;

-- name: CreateUnit :one
INSERT INTO units (name_en, name_de, abbreviation_en, abbreviation_de, plural_en, plural_de)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: UpdateUnit :one
UPDATE units SET
  name_en = COALESCE($2, name_en),
  name_de = COALESCE($3, name_de),
  abbreviation_en = COALESCE($4, abbreviation_en),
  abbreviation_de = COALESCE($5, abbreviation_de)
WHERE id = $1 RETURNING *;

-- name: MergeUnit :exec
UPDATE step_ingredients SET unit_id = $2 WHERE unit_id = $1;

-- name: DeleteUnit :exec
DELETE FROM units WHERE id = $1;

-- name: ListUnits :many
SELECT * FROM units ORDER BY name_en LIMIT $1 OFFSET $2;

-- name: CountUnits :one
SELECT COUNT(*) FROM units;
```

**Step 6: Run sqlc generate**

```bash
cd backend && sqlc generate
```

Expected: generated Go files in `internal/db/queries/`

**Step 7: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(infra): add sqlc query definitions and generate db layer"
```

---

## Task 6: Auth API

**Files:**
- Create: `backend/internal/auth/jwt.go`
- Create: `backend/internal/auth/password.go`
- Create: `backend/internal/auth/passkey.go`
- Create: `backend/internal/api/auth_handler.go`
- Create: `backend/internal/api/auth_handler_test.go`
- Create: `backend/internal/api/routes.go`
- Create: `backend/internal/api/middleware.go`

**Step 1: jwt.go**

```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(secret string, userID uuid.UUID, username, role string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateAccessToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
```

**Step 2: password.go**

```go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
```

**Step 3: Write auth handler tests first (TDD)**

`backend/internal/api/auth_handler_test.go` — test login, register, refresh, logout endpoints with table-driven tests using httptest.

**Step 4: auth_handler.go**

Implement handlers:
- `POST /api/auth/setup` — first-run admin creation (only when setup_complete=false)
- `POST /api/auth/register` — register (requires open_registration=true OR valid invite token)
- `POST /api/auth/login` — username+password → access token + refresh token cookie
- `POST /api/auth/refresh` — refresh token cookie → new access token
- `POST /api/auth/logout` — delete refresh token
- `GET /api/auth/me` — return current user (requires auth)
- `GET /api/auth/passkey/register/begin` — begin WebAuthn registration
- `POST /api/auth/passkey/register/finish` — complete WebAuthn registration
- `GET /api/auth/passkey/login/begin` — begin WebAuthn login
- `POST /api/auth/passkey/login/finish` — complete WebAuthn login

**Step 5: middleware.go**

```go
package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/auth"
)

func (s *Server) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Request().Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ValidateAccessToken(s.cfg.JWTSecret, token)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		return next(c)
	}
}

func (s *Server) AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		role, _ := c.Get("role").(string)
		if role != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, "admin required")
		}
		return next(c)
	}
}
```

**Step 6: routes.go**

```go
package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/config"
)

type Server struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func RegisterRoutes(e *echo.Echo, pool *pgxpool.Pool, cfg *config.Config) {
	s := &Server{pool: pool, cfg: cfg}

	// Static file serving for uploads
	e.Static("/api/media", cfg.UploadPath)

	// Health check
	e.GET("/api/health", s.HealthCheck)

	// Auth routes (public)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/setup", s.SetupAdmin)
	authGroup.POST("/register", s.Register)
	authGroup.POST("/login", s.Login)
	authGroup.POST("/refresh", s.RefreshToken)
	authGroup.POST("/logout", s.Logout)
	authGroup.GET("/passkey/register/begin", s.PasskeyRegisterBegin, s.AuthMiddleware)
	authGroup.POST("/passkey/register/finish", s.PasskeyRegisterFinish, s.AuthMiddleware)
	authGroup.GET("/passkey/login/begin", s.PasskeyLoginBegin)
	authGroup.POST("/passkey/login/finish", s.PasskeyLoginFinish)

	// Protected routes
	api := e.Group("/api", s.AuthMiddleware)
	api.GET("/auth/me", s.GetCurrentUser)

	// Recipe routes
	api.GET("/recipes", s.ListRecipes)
	api.POST("/recipes", s.CreateRecipe)
	api.GET("/recipes/:id", s.GetRecipe)
	api.PUT("/recipes/:id", s.UpdateRecipe)
	api.DELETE("/recipes/:id", s.DeleteRecipe)
	api.POST("/recipes/:id/photo", s.UploadRecipePhoto)
	api.GET("/recipes/:id/steps", s.ListRecipeSteps)
	api.POST("/recipes/:id/steps", s.CreateRecipeStep)
	api.PUT("/recipes/:id/steps/:stepId", s.UpdateRecipeStep)
	api.DELETE("/recipes/:id/steps/:stepId", s.DeleteRecipeStep)
	api.POST("/recipes/:id/steps/:stepId/ingredients", s.AddStepIngredient)
	api.DELETE("/recipes/:id/steps/:stepId/ingredients/:ingId", s.RemoveStepIngredient)
	api.POST("/recipes/:id/share", s.CreateShareLink)

	// Ingredient routes
	api.GET("/ingredients", s.ListIngredients)
	api.POST("/ingredients", s.CreateIngredient)
	api.GET("/ingredients/:id", s.GetIngredient)
	api.PUT("/ingredients/:id", s.UpdateIngredient)
	api.POST("/ingredients/merge", s.MergeIngredients)

	// Unit routes
	api.GET("/units", s.ListUnits)
	api.POST("/units", s.CreateUnit)
	api.GET("/units/:id", s.GetUnit)
	api.PUT("/units/:id", s.UpdateUnit)
	api.POST("/units/merge", s.MergeUnits)

	// Admin routes
	admin := e.Group("/api/admin", s.AuthMiddleware, s.AdminMiddleware)
	admin.GET("/users", s.ListUsers)
	admin.POST("/users", s.CreateUserAdmin)
	admin.DELETE("/users/:id", s.DeleteUser)
	admin.POST("/invites", s.CreateInviteLink)
	admin.GET("/invites", s.ListInviteLinks)
	admin.DELETE("/invites/:id", s.DeleteInviteLink)
	admin.GET("/settings", s.GetSettings)
	admin.PUT("/settings", s.UpdateSettings)

	// Public routes (no auth)
	public := e.Group("/api/public")
	public.GET("/recipes", s.ListPublicRecipes)
	public.GET("/recipes/:id", s.GetPublicRecipe)
	public.GET("/share/:token", s.GetSharedRecipes)

	// kitshn-compatible compat (for kitshn)
	kitshn-compat := e.Group("/api")
	kitshn-compat.GET("/recipe/", s.kitshn-compatibleListRecipes, s.AuthMiddleware)
	kitshn-compat.GET("/recipe/:id/", s.kitshn-compatibleGetRecipe, s.AuthMiddleware)
	kitshn-compat.POST("/recipe/", s.kitshn-compatibleCreateRecipe, s.AuthMiddleware)
	kitshn-compat.GET("/food/", s.kitshn-compatibleListFood, s.AuthMiddleware)
	kitshn-compat.GET("/unit/", s.kitshn-compatibleListUnits, s.AuthMiddleware)
	kitshn-compat.GET("/keyword/", s.kitshn-compatibleListKeywords, s.AuthMiddleware)
}
```

**Step 7: Run tests**

```bash
cd backend && go test ./internal/api/... -v
```

**Step 8: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(auth): implement JWT auth, passkeys, registration with invite links"
```

---

## Task 7: Recipe, Ingredient, Unit API Handlers

**Files:**
- Create: `backend/internal/api/recipe_handler.go`
- Create: `backend/internal/api/recipe_handler_test.go`
- Create: `backend/internal/api/ingredient_handler.go`
- Create: `backend/internal/api/unit_handler.go`
- Create: `backend/internal/api/admin_handler.go`
- Create: `backend/internal/api/kitshn-compat_handler.go`
- Create: `backend/internal/api/public_handler.go`
- Create: `backend/internal/storage/storage.go`

**Step 1: storage.go for file uploads**

```go
package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Storage struct {
	BasePath string
}

func New(basePath string) *Storage {
	os.MkdirAll(basePath, 0755)
	return &Storage{BasePath: basePath}
}

func (s *Storage) SavePhoto(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	filename := uuid.NewString() + ext
	dst, err := os.Create(filepath.Join(s.BasePath, filename))
	if err != nil {
		return "", err
	}
	defer dst.Close()
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return filename, nil
}

func (s *Storage) DeletePhoto(filename string) error {
	path := filepath.Join(s.BasePath, filepath.Base(filename))
	return os.Remove(path)
}
```

**Step 2: recipe_handler.go**

Implement all recipe CRUD handlers. Each handler:
1. Parse and validate request body
2. Call sqlc query via pool
3. Return JSON response with appropriate status code
4. Update ingredient_unit_stats when ingredients are added to steps

**Step 3: ingredient_handler.go**

- `GET /api/ingredients?search=X&limit=20` — fuzzy search with trigram index
- `POST /api/ingredients` — create
- `PUT /api/ingredients/:id` — update  
- `POST /api/ingredients/merge` — body: `{source_id, target_id}` — moves all references to target, deletes source
- On search: also return `suggested_unit` from ingredient_unit_stats

**Step 4: unit_handler.go**

Same pattern as ingredients.

**Step 5: kitshn-compat_handler.go**

Map Wokko data models to kitshn-compatible v1 API response format. kitshn-compatible uses specific JSON shapes — implement the minimal subset kitshn needs:

```go
// kitshn-compatible Recipe response shape
type kitshn-compatibleRecipe struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
	WorkingTime int    `json:"working_time"`
	WaitingTime int    `json:"waiting_time"`
	Servings    int    `json:"servings"`
	Keywords    []kitshn-compatibleKeyword `json:"keywords"`
	Steps       []kitshn-compatibleStep    `json:"steps"`
}
```

Note: kitshn-compatible uses integer IDs. Map UUID to a deterministic int by using a hash or sequential counter. Store a `kitshn-compat_id_seq` counter in system_settings.

**Step 6: Write tests for recipe handler**

Test list, create, get, update, delete with mock DB or test database.

**Step 7: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(recipes): implement recipe CRUD, ingredient/unit handlers, kitshn-compatible compat"
```

---

## Task 8: Docker Compose

**Files:**
- Create: `docker-compose.yml`
- Create: `docker-compose.dev.yml`
- Create: `.env.example`

**Step 1: docker-compose.yml**

```yaml
name: wokko

services:
  db:
    image: postgres:16-alpine
    container_name: wokko-db
    restart: unless-stopped
    environment:
      POSTGRES_DB: wokko
      POSTGRES_USER: wokko
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-changeme}
    volumes:
      - wokko-db:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U wokko"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    image: ghcr.io/serraniel/wokko-backend:${WOKKO_VERSION:-latest}
    container_name: wokko-backend
    restart: unless-stopped
    depends_on:
      db:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://wokko:${POSTGRES_PASSWORD:-changeme}@db:5432/wokko?sslmode=disable
      JWT_SECRET: ${JWT_SECRET}
      PUBLIC_URL: ${PUBLIC_URL:-http://localhost:3000}
      UPLOAD_PATH: /data/uploads
      PORT: 8080
      OPEN_REGISTRATION: ${OPEN_REGISTRATION:-false}
      ALLOW_PUBLIC_RECIPES: ${ALLOW_PUBLIC_RECIPES:-true}
      MCP_ENABLED: ${MCP_ENABLED:-false}
      RP_ID: ${RP_ID:-localhost}
      RP_ORIGIN: ${RP_ORIGIN:-http://localhost:3000}
      RP_DISPLAY_NAME: ${RP_DISPLAY_NAME:-Wokko}
    volumes:
      - wokko-uploads:/data/uploads
    ports:
      - "127.0.0.1:8080:8080"

  frontend:
    image: ghcr.io/serraniel/wokko-frontend:${WOKKO_VERSION:-latest}
    container_name: wokko-frontend
    restart: unless-stopped
    environment:
      PUBLIC_API_URL: ${PUBLIC_API_URL:-http://backend:8080}
      ORIGIN: ${PUBLIC_URL:-http://localhost:3000}
    ports:
      - "127.0.0.1:3000:3000"

volumes:
  wokko-db:
    name: wokko-db
  wokko-uploads:
    name: wokko-uploads
```

**Step 2: docker-compose.dev.yml**

Override for local development with hot reload and direct DB access:

```yaml
services:
  db:
    ports:
      - "5432:5432"

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    volumes:
      - ./backend:/app
    environment:
      DATABASE_URL: postgres://wokko:changeme@db:5432/wokko?sslmode=disable
      JWT_SECRET: dev-secret-change-in-production

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    volumes:
      - ./frontend:/app
      - /app/node_modules
    environment:
      PUBLIC_API_URL: http://backend:8080
```

**Step 3: .env.example**

```bash
# Database
POSTGRES_PASSWORD=your-secure-password-here

# JWT (generate with: openssl rand -hex 32)
JWT_SECRET=your-jwt-secret-here

# Application
PUBLIC_URL=https://wokko.yourdomain.com
PUBLIC_API_URL=http://backend:8080

# WebAuthn (set to your domain)
RP_ID=wokko.yourdomain.com
RP_ORIGIN=https://wokko.yourdomain.com
RP_DISPLAY_NAME=Wokko

# Feature flags
OPEN_REGISTRATION=false
ALLOW_PUBLIC_RECIPES=true
MCP_ENABLED=false

# Docker image version (set automatically by CI)
WOKKO_VERSION=latest
```

**Step 4: Commit**

```bash
git -c commit.gpgsign=false commit -m "chore(infra): add Docker Compose configuration with named volumes"
```

---

## Task 9: GitHub Actions CI/CD

**Files:**
- Create: `.github/workflows/pr-check.yml`
- Create: `.github/workflows/nightly.yml`
- Create: `.github/workflows/docker-publish.yml`
- Create: `.github/dependabot.yml`

**Step 1: pr-check.yml**

```yaml
name: PR Check

on:
  pull_request:
    branches: [main]

jobs:
  backend:
    name: Backend — Build & Test
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: wokko_test
          POSTGRES_USER: wokko
          POSTGRES_PASSWORD: testpassword
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache-dependency-path: backend/go.sum
      - name: Build
        working-directory: backend
        run: go build ./...
      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          working-directory: backend
      - name: Test
        working-directory: backend
        env:
          DATABASE_URL: postgres://wokko:testpassword@localhost:5432/wokko_test?sslmode=disable
          JWT_SECRET: test-secret
        run: go test ./... -v -cover -coverprofile=coverage.out
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: backend/coverage.out

  frontend:
    name: Frontend — Build & Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json
      - name: Install
        working-directory: frontend
        run: npm ci
      - name: Lint
        working-directory: frontend
        run: npm run lint
      - name: Type check
        working-directory: frontend
        run: npm run check
      - name: Test
        working-directory: frontend
        run: npm test
      - name: Build
        working-directory: frontend
        env:
          VITE_APP_VERSION: pr-check
        run: npm run build
```

**Step 2: nightly.yml**

```yaml
name: Nightly Release

on:
  schedule:
    - cron: '0 2 * * *'  # 2am UTC daily
  workflow_dispatch:

jobs:
  check-changes:
    runs-on: ubuntu-latest
    outputs:
      has_changes: ${{ steps.check.outputs.has_changes }}
      new_version: ${{ steps.version.outputs.new_version }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Check for changes since last release
        id: check
        run: |
          LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
          if [ -z "$LAST_TAG" ]; then
            echo "has_changes=true" >> $GITHUB_OUTPUT
          else
            CHANGES=$(git log $LAST_TAG..HEAD --oneline | wc -l)
            if [ "$CHANGES" -gt 0 ]; then
              echo "has_changes=true" >> $GITHUB_OUTPUT
            else
              echo "has_changes=false" >> $GITHUB_OUTPUT
            fi
          fi
      - name: Calculate next version
        id: version
        run: |
          LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
          # Bump patch version
          VERSION=$(echo $LAST_TAG | sed 's/v//')
          PATCH=$(echo $VERSION | cut -d. -f3)
          NEW_PATCH=$((PATCH + 1))
          NEW_VERSION="$(echo $VERSION | cut -d. -f1-2).$NEW_PATCH"
          echo "new_version=$NEW_VERSION" >> $GITHUB_OUTPUT

  release:
    needs: check-changes
    if: needs.check-changes.outputs.has_changes == 'true'
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Generate changelog
        id: changelog
        run: |
          LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
          if [ -z "$LAST_TAG" ]; then
            RANGE="HEAD"
          else
            RANGE="$LAST_TAG..HEAD"
          fi
          
          # Group PRs by label (use commit message scope as category)
          generate_section() {
            local scope=$1
            local title=$2
            local commits=$(git log $RANGE --oneline --grep="($scope)" 2>/dev/null || true)
            if [ -n "$commits" ]; then
              echo "### $title"
              echo "$commits" | while read line; do echo "- $line"; done
              echo ""
            fi
          }
          
          CHANGELOG="## What's Changed\n\n"
          CHANGELOG+=$(generate_section "auth" "Auth & Users")
          CHANGELOG+=$(generate_section "recipes" "Recipes")
          CHANGELOG+=$(generate_section "ingredients" "Ingredients")
          CHANGELOG+=$(generate_section "units" "Units")
          CHANGELOG+=$(generate_section "sharing" "Sharing")
          CHANGELOG+=$(generate_section "mcp" "MCP")
          CHANGELOG+=$(generate_section "i18n" "Internationalization")
          CHANGELOG+=$(generate_section "ui" "UI")
          CHANGELOG+=$(generate_section "infra" "Infrastructure")
          CHANGELOG+=$(generate_section "fix" "Bug Fixes")
          
          echo "changelog<<EOF" >> $GITHUB_OUTPUT
          echo -e "$CHANGELOG" >> $GITHUB_OUTPUT
          echo "EOF" >> $GITHUB_OUTPUT

      - name: Create release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: v${{ needs.check-changes.outputs.new_version }}
          name: v${{ needs.check-changes.outputs.new_version }} (nightly)
          body: ${{ steps.changelog.outputs.changelog }}
          prerelease: true
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  docker:
    needs: [check-changes, release]
    if: needs.check-changes.outputs.has_changes == 'true'
    uses: ./.github/workflows/docker-publish.yml
    with:
      version: ${{ needs.check-changes.outputs.new_version }}
    secrets: inherit
```

**Step 3: docker-publish.yml**

```yaml
name: Docker Publish

on:
  workflow_call:
    inputs:
      version:
        required: true
        type: string
  release:
    types: [published]

jobs:
  build-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    strategy:
      matrix:
        include:
          - context: ./backend
            image: ghcr.io/serraniel/wokko-backend
          - context: ./frontend
            image: ghcr.io/serraniel/wokko-frontend
    steps:
      - uses: actions/checkout@v4
      - name: Set version
        id: ver
        run: |
          if [ -n "${{ inputs.version }}" ]; then
            echo "version=${{ inputs.version }}" >> $GITHUB_OUTPUT
          else
            echo "version=${GITHUB_REF_NAME#v}" >> $GITHUB_OUTPUT
          fi
      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      - name: Docker meta
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ matrix.image }}
          tags: |
            type=raw,value=latest
            type=raw,value=${{ steps.ver.outputs.version }}
            type=semver,pattern={{major}}.{{minor}},value=${{ steps.ver.outputs.version }}
            type=semver,pattern={{major}},value=${{ steps.ver.outputs.version }}
      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: ${{ matrix.context }}
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          platforms: linux/amd64,linux/arm64
          build-args: |
            VITE_APP_VERSION=${{ steps.ver.outputs.version }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

**Step 4: dependabot.yml**

```yaml
version: 2
updates:
  # GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "daily"
    labels: ["Infra"]

  # Go backend
  - package-ecosystem: "gomod"
    directory: "/backend"
    schedule:
      interval: "daily"
    labels: ["Infra"]

  # npm frontend
  - package-ecosystem: "npm"
    directory: "/frontend"
    schedule:
      interval: "daily"
    labels: ["Infra"]

  # Docker base images
  - package-ecosystem: "docker"
    directory: "/backend"
    schedule:
      interval: "daily"
    labels: ["Infra"]
  - package-ecosystem: "docker"
    directory: "/frontend"
    schedule:
      interval: "daily"
    labels: ["Infra"]
  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "daily"
    labels: ["Infra"]
```

**Step 5: Commit**

```bash
git -c commit.gpgsign=false commit -m "chore(ci): add GitHub Actions workflows and Dependabot configuration"
```

---

## Task 10: Frontend Scaffold

**Files:**
- Create: `frontend/` — full SvelteKit project

**Step 1: Initialize SvelteKit**

```bash
cd /home/serraniel/Projects/Wokko
npm create svelte@latest frontend -- --template skeleton --types typescript --no-prettier --no-eslint --no-playwright --no-vitest
cd frontend
npm install
```

**Step 2: Add dependencies**

```bash
npm install -D tailwindcss @tailwindcss/vite
npm install -D @tailwindcss/typography
npm install @paraglide/sveltekit-paraglide
npm install bits-ui
npm install clsx tailwind-merge
npm install lucide-svelte
npm install @tanstack/svelte-query
npm install svelte-sonner
npm install @internationalized/date
npm install -D vitest @testing-library/svelte @testing-library/jest-dom
npm install -D eslint eslint-plugin-svelte @typescript-eslint/eslint-plugin
```

**Step 3: Configure TailwindCSS v4**

`frontend/src/app.css`:
```css
@import 'tailwindcss';

@theme {
  --color-wokko-red: #C0392B;
  --color-wokko-orange: #E67E22;
  --color-wokko-cream: #FFF8F0;
  --font-sans: 'Inter', ui-sans-serif, system-ui, sans-serif;
}
```

`frontend/vite.config.ts`:
```typescript
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	define: {
		'import.meta.env.VITE_APP_VERSION': JSON.stringify(process.env.VITE_APP_VERSION || 'dev')
	}
});
```

**Step 4: Configure Paraglide i18n**

`frontend/project.inlang/settings.json`:
```json
{
	"$schema": "https://inlang.com/schema/project-settings",
	"sourceLanguageTag": "en",
	"languageTags": ["en", "de"],
	"modules": [
		"https://cdn.jsdelivr.net/npm/@inlang/plugin-message-format/dist/index.js",
		"https://cdn.jsdelivr.net/npm/@inlang/paraglide-js-adapter-sveltekit/dist/index.js"
	]
}
```

Create `frontend/messages/en.json` and `frontend/messages/de.json` with all UI strings.

**Step 5: Create API client**

`frontend/src/lib/api/client.ts` — typed fetch wrapper that reads `PUBLIC_API_URL` and adds auth headers from localStorage/cookie.

**Step 6: Frontend Dockerfile**

`frontend/Dockerfile`:
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_APP_VERSION=dev
ENV VITE_APP_VERSION=$VITE_APP_VERSION
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/build ./build
COPY --from=builder /app/package*.json ./
RUN npm ci --production
EXPOSE 3000
CMD ["node", "build"]
```

**Step 7: Commit**

```bash
git -c commit.gpgsign=false commit -m "chore(ui): scaffold SvelteKit frontend with TailwindCSS and Paraglide i18n"
```

---

## Task 11: Frontend Pages

**Files:**
- Create: `frontend/src/routes/+layout.svelte`
- Create: `frontend/src/routes/+layout.ts`
- Create: `frontend/src/routes/setup/+page.svelte`
- Create: `frontend/src/routes/auth/login/+page.svelte`
- Create: `frontend/src/routes/auth/register/+page.svelte`
- Create: `frontend/src/routes/(app)/+layout.svelte`
- Create: `frontend/src/routes/(app)/recipes/+page.svelte`
- Create: `frontend/src/routes/(app)/recipes/new/+page.svelte`
- Create: `frontend/src/routes/(app)/recipes/[id]/+page.svelte`
- Create: `frontend/src/routes/(app)/recipes/[id]/edit/+page.svelte`
- Create: `frontend/src/routes/(app)/admin/+page.svelte`
- Create: `frontend/src/routes/(public)/+page.svelte`
- Create: `frontend/src/lib/components/Footer.svelte`
- Create: `frontend/src/lib/components/RecipeCard.svelte`
- Create: `frontend/src/lib/components/IngredientInput.svelte`
- Create: `frontend/src/lib/components/UnitInput.svelte`
- Create: `frontend/src/lib/components/ThemeToggle.svelte`
- Create: `frontend/src/lib/components/Logo.svelte`

**Step 1: Root layout with theme and nav**

`+layout.svelte` sets up HTML structure, dark mode class toggle, navigation sidebar/header, and Footer.

**Step 2: Setup wizard page**

`/setup` — shown when `setup_complete=false`. Multi-step form:
1. Welcome to Wokko
2. Create admin account (username, password, optional email)
3. Basic settings (public URL, open registration toggle)
4. Done + redirect to login

**Step 3: Auth pages**

`/auth/login` — username/password form + "Login with Passkey" button
`/auth/register` — username, password, optional email, optional invite token from URL query param

**Step 4: Recipe list page**

`/recipes` — grid of RecipeCard components with search, tag filter, "New Recipe" button.

**Step 5: Recipe editor page**

`/recipes/new` and `/recipes/[id]/edit`:
- Recipe name, description, tags (multi-select with autocomplete)
- Work time, servings
- Photo upload with preview
- Step list: ordered, each step has:
  - Description (textarea)
  - Optional duration
  - Ingredients list (each: IngredientInput + UnitInput + amount + notes)
  - Add/remove ingredient buttons
- Add/remove steps
- Save button with loading state

**Step 6: IngredientInput component**

Combobox that:
- Shows dropdown with fuzzy-matched existing ingredients as user types
- Option at bottom: "Create '[typed text]'"
- On selection, auto-fills the unit field with the ingredient's suggested default unit

**Step 7: Footer component**

```svelte
<footer class="border-t py-4 text-sm text-muted-foreground">
  <div class="flex items-center justify-center gap-3">
    <a href="https://github.com/Serraniel/Wokko" target="_blank">GitHub</a>
    <span>·</span>
    <a href="https://github.com/Serraniel/Wokko/releases/tag/v{version}" target="_blank">
      v{version}
    </a>
    <span>·</span>
    <a href="/changelog">Changelog</a>
    <span>·</span>
    <a href="/license">EUPL-1.2</a>
    <span>·</span>
    <a href="/third-party">Third-Party Notices</a>
  </div>
</footer>
```

Version is `import.meta.env.VITE_APP_VERSION`.

**Step 8: Admin panel**

`/admin` — tabs for:
- Users: table with username, role, actions (delete, promote)
- Invite Links: generate link button, list with token/expiry/usage, copy and revoke
- Settings: toggle open registration, toggle public recipes, MCP enabled

**Step 9: Public gallery**

`/(public)` — `/` route shows public recipe gallery when not logged in (or always if `ALLOW_PUBLIC_RECIPES=true`). Grid of public recipes, no login required.

**Step 10: Recipe detail page**

`/recipes/[id]` — full recipe view:
- Hero photo
- Name, description, tags, work time, servings
- Ingredients overview (all ingredients aggregated from all steps)
- Steps with their specific ingredients
- Share button (generates share link)

**Step 11: Build and test**

```bash
cd frontend && npm run build
npm run check
npm test
```

**Step 12: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(ui): implement all frontend pages including recipe editor and admin panel"
```

---

## Task 12: SVG Logo

**Files:**
- Create: `frontend/static/logo.svg`
- Create: `frontend/static/favicon.svg`
- Create: `frontend/static/favicon.ico` (derived from SVG)
- Modify: `frontend/src/app.html` — use favicon
- Create: `frontend/src/lib/components/Logo.svelte`

**Step 1: Create logo.svg**

Simplified red wok character based on the Gemini drafts:
- Round red pot body `#C0392B` with small handles on sides
- Cute face: large round eyes, simple smile
- Orange-red flame on top `#E67E22`
- Clean, flat design suitable for app icon

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 120" fill="none">
  <!-- Flame -->
  <path d="M50 5 C45 15 35 20 38 35 C30 28 32 18 28 15 C22 25 25 40 35 45 C30 42 28 48 30 52 C35 48 45 50 50 45 C55 50 65 48 70 52 C72 48 70 42 65 45 C75 40 78 25 72 15 C68 18 70 28 62 35 C65 20 55 15 50 5Z" fill="#E67E22"/>
  <path d="M50 10 C47 18 41 22 43 33 C38 28 40 20 37 17 C33 25 35 37 42 41 C39 39 38 43 40 46 C43 43 47 44 50 42 C53 44 57 43 60 46 C62 43 61 39 58 41 C65 37 67 25 63 17 C60 20 62 28 57 33 C59 22 53 18 50 10Z" fill="#F39C12"/>
  <!-- Wok body -->
  <ellipse cx="50" cy="75" rx="35" ry="30" fill="#C0392B"/>
  <ellipse cx="50" cy="72" rx="32" ry="27" fill="#E74C3C"/>
  <!-- Handles -->
  <rect x="12" y="68" width="10" height="8" rx="4" fill="#A93226"/>
  <rect x="78" y="68" width="10" height="8" rx="4" fill="#A93226"/>
  <!-- Face - eyes -->
  <circle cx="40" cy="68" r="5" fill="white"/>
  <circle cx="60" cy="68" r="5" fill="white"/>
  <circle cx="41" cy="69" r="3" fill="#1a1a1a"/>
  <circle cx="61" cy="69" r="3" fill="#1a1a1a"/>
  <!-- Eye shine -->
  <circle cx="42" cy="67" r="1" fill="white"/>
  <circle cx="62" cy="67" r="1" fill="white"/>
  <!-- Smile -->
  <path d="M41 80 Q50 88 59 80" stroke="#A93226" stroke-width="2.5" stroke-linecap="round" fill="none"/>
  <!-- Bottom legs/feet -->
  <ellipse cx="40" cy="103" rx="6" ry="4" fill="#A93226"/>
  <ellipse cx="60" cy="103" rx="6" ry="4" fill="#A93226"/>
  <rect x="37" y="98" width="6" height="8" rx="2" fill="#C0392B"/>
  <rect x="57" y="98" width="6" height="8" rx="2" fill="#C0392B"/>
</svg>
```

**Step 2: Create favicon.svg** (simplified 32x32 version — just the wok head)

**Step 3: Logo.svelte component**

```svelte
<script lang="ts">
  let { size = 40, class: className = '' } = $props();
</script>

<img src="/logo.svg" alt="Wokko" width={size} height={size} class={className} />
```

**Step 4: Update app.html**

```html
<link rel="icon" href="%sveltekit.assets%/favicon.svg" type="image/svg+xml">
<link rel="icon" href="%sveltekit.assets%/favicon.ico" sizes="any">
```

**Step 5: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(ui): add Wokko logo SVG and favicon"
```

---

## Task 13: Translations (EN + DE)

**Files:**
- Create: `frontend/messages/en.json`
- Create: `frontend/messages/de.json`

**Step 1: en.json** — all UI strings in English:

Cover: navigation items, auth forms (login, register, setup), recipe editor labels (name, description, steps, ingredients, units, amounts, notes, time, servings, tags, photo), admin panel, settings, share/public, error messages, empty states, loading states, footer.

**Step 2: de.json** — German translations for all keys in en.json

**Step 3: Run Paraglide compile**

```bash
cd frontend && npx @inlang/paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide
```

**Step 4: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(i18n): add English and German translations"
```

---

## Task 14: Backend Tests

**Files:**
- Create: `backend/internal/api/testhelpers_test.go`
- Create: `backend/internal/auth/jwt_test.go`
- Create: `backend/internal/auth/password_test.go`
- Create: `backend/internal/api/auth_handler_test.go`
- Create: `backend/internal/api/recipe_handler_test.go`
- Create: `backend/internal/api/ingredient_handler_test.go`

**Step 1: testhelpers_test.go**

Test helper that spins up a real PostgreSQL connection (uses `DATABASE_URL` env, defaults to a test DB). Provides:
- `setupTestDB(t)` — creates tables, returns pool + cleanup func
- `createTestUser(t, pool, role)` — creates a user, returns user + JWT
- `newTestServer(t, pool, cfg)` — returns httptest.Server with all routes

**Step 2: auth tests**

Test JWT generation/validation, password hashing, login endpoint, register with/without invite.

**Step 3: recipe tests**

Test CRUD, auth required for mutations, ownership enforcement (user can only edit own recipes), photo upload.

**Step 4: ingredient tests**

Test search with fuzzy matching, create, merge (verifies references are updated).

**Step 5: Run all tests**

```bash
cd backend && go test ./... -v
```

Expected: all PASS

**Step 6: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(infra): add comprehensive API integration tests"
```

---

## Task 15: Seed Data (Full EN+DE)

**File:** `backend/migrations/000002_seed_data.up.sql`

Write complete seed SQL with:

**Units (30+):**
| name_en | name_de | abbr_en | abbr_de | plural_en | plural_de |
|---------|---------|---------|---------|-----------|-----------|
| gram | Gramm | g | g | grams | Gramm |
| kilogram | Kilogramm | kg | kg | kilograms | Kilogramm |
| milligram | Milligramm | mg | mg | milligrams | Milligramm |
| milliliter | Milliliter | ml | ml | milliliters | Milliliter |
| liter | Liter | l | l | liters | Liter |
| centiliter | Zentiliter | cl | cl | centiliters | Zentiliter |
| teaspoon | Teelöffel | tsp | TL | teaspoons | Teelöffel |
| tablespoon | Esslöffel | tbsp | EL | tablespoons | Esslöffel |
| cup | Tasse | cup | Tasse | cups | Tassen |
| fluid ounce | Flüssigkeitsunze | fl oz | fl oz | fluid ounces | Flüssigkeitsunzen |
| ounce | Unze | oz | oz | ounces | Unzen |
| pound | Pfund | lb | Pfd | pounds | Pfund |
| piece | Stück | pcs | Stk | pieces | Stücke |
| pinch | Prise | pinch | Prise | pinches | Prisen |
| bunch | Bund | bunch | Bund | bunches | Bunde |
| clove | Zehe | clove | Zehe | cloves | Zehen |
| can | Dose | can | Dose | cans | Dosen |
| package | Packung | pkg | Pkg | packages | Packungen |
| slice | Scheibe | slice | Scheibe | slices | Scheiben |
| drop | Tropfen | drop | Tropfen | drops | Tropfen |
| handful | Handvoll | handful | Handvoll | handfuls | Handvoll |
| dash | Schuss | dash | Schuss | dashes | Schüsse |
| stalk | Stange | stalk | Stange | stalks | Stangen |
| head | Kopf | head | Kopf | heads | Köpfe |
| sheet | Blatt | sheet | Blatt | sheets | Blätter |
| jar | Glas | jar | Glas | jars | Gläser |
| bottle | Flasche | bottle | Flasche | bottles | Flaschen |
| bag | Beutel | bag | Beutel | bags | Beutel |
| cube | Würfel | cube | Würfel | cubes | Würfel |
| sprig | Zweig | sprig | Zweig | sprigs | Zweige |

**Ingredients (200+) organized by category:**

Vegetables: onion/Zwiebel, garlic/Knoblauch, carrot/Karotte, potato/Kartoffel, tomato/Tomate, cucumber/Gurke, bell pepper/Paprika, broccoli/Brokkoli, spinach/Spinat, lettuce/Salat, zucchini/Zucchini, eggplant/Aubergine, cauliflower/Blumenkohl, cabbage/Kohl, leek/Lauch, celery/Sellerie, mushroom/Champignon, peas/Erbsen, corn/Mais, asparagus/Spargel, artichoke/Artischocke, pumpkin/Kürbis, sweet potato/Süßkartoffel, radish/Radieschen, beet/Rote Bete...

Fruits: apple/Apfel, banana/Banane, orange/Orange, lemon/Zitrone, lime/Limette, strawberry/Erdbeere, blueberry/Heidelbeere, raspberry/Himbeere, grape/Weintraube, mango/Mango, pineapple/Ananas, watermelon/Wassermelone, peach/Pfirsich, pear/Birne, cherry/Kirsche, plum/Pflaume, apricot/Aprikose, kiwi/Kiwi, avocado/Avocado, coconut/Kokosnuss...

Proteins: chicken breast/Hähnchenbrust, ground beef/Hackfleisch, salmon/Lachs, egg/Ei, tofu/Tofu, lentils/Linsen, chickpeas/Kichererbsen, black beans/Schwarze Bohnen, kidney beans/Kidneybohnen, shrimp/Garnelen, tuna/Thunfisch, pork/Schweinefleisch, beef/Rindfleisch, lamb/Lammfleisch, turkey/Pute, cod/Kabeljau...

Dairy: milk/Milch, butter/Butter, cream/Sahne, yogurt/Joghurt, mozzarella/Mozzarella, parmesan/Parmesan, cheddar/Cheddar, feta/Feta, cream cheese/Frischkäse, sour cream/Saure Sahne, heavy cream/Schlagsahne, gouda/Gouda...

Grains & Pasta: all-purpose flour/Mehl, bread crumbs/Paniermehl, rice/Reis, pasta/Pasta, spaghetti/Spaghetti, penne/Penne, bread/Brot, oats/Haferflocken, quinoa/Quinoa, couscous/Couscous, breadcrumbs/Semmelbrösel, cornstarch/Speisestärke...

Spices & Herbs: salt/Salz, black pepper/Schwarzer Pfeffer, paprika/Paprikapulver, cumin/Kreuzkümmel, coriander/Koriander, turmeric/Kurkuma, cinnamon/Zimt, oregano/Oregano, basil/Basilikum, thyme/Thymian, rosemary/Rosmarin, bay leaf/Lorbeerblatt, parsley/Petersilie, chili flakes/Chiliflocken, garlic powder/Knoblauchpulver, onion powder/Zwiebelpulver, ginger/Ingwer, nutmeg/Muskatnuss, cardamom/Kardamom, cloves/Gewürznelken, vanilla/Vanille...

Liquids & Sauces: olive oil/Olivenöl, vegetable oil/Pflanzenöl, soy sauce/Sojasoße, vinegar/Essig, balsamic vinegar/Balsamicoessig, honey/Honig, maple syrup/Ahornsirup, tomato paste/Tomatenmark, coconut milk/Kokosmilch, broth/Brühe, wine/Wein, lemon juice/Zitronensaft...

Baking: sugar/Zucker, brown sugar/Brauner Zucker, powdered sugar/Puderzucker, baking powder/Backpulver, baking soda/Natron, cocoa powder/Kakaopulver, dark chocolate/Zartbitterschokolade, vanilla extract/Vanilleextrakt, yeast/Hefe, almond flour/Mandelmehl...

**Tags (20+):** Vegetarian, Vegan, Gluten-Free, Dairy-Free, Low-Carb, Quick (<30min), Comfort Food, Breakfast, Lunch, Dinner, Dessert, Snack, Soup, Salad, Asian, Italian, German, Mexican, Indian, BBQ

**Step 1: Write the full seed SQL**

**Step 2: Test migration runs successfully**

```bash
cd backend && DATABASE_URL="postgres://wokko:changeme@localhost:5432/wokko_test?sslmode=disable" go run ./cmd/server
```

Check that seeds are inserted.

**Step 3: Commit**

```bash
git -c commit.gpgsign=false commit -m "feat(ingredients): add comprehensive seed data for units, ingredients, and tags (EN+DE)"
```

---

## Task 16: Final Integration & Push

**Step 1: Build everything**

```bash
cd backend && go build ./...
cd ../frontend && npm run build
```

**Step 2: Run all tests**

```bash
cd backend && go test ./... -v
cd ../frontend && npm test
```

**Step 3: Tag initial version**

```bash
git tag -a v0.1.0 -m "v0.1.0 — Initial release"
git push origin main --tags
```

**Step 4: Create v0.1.0 GitHub release**

```bash
gh release create v0.1.0 \
  --title "v0.1.0 — Foundation" \
  --notes "Initial release of Wokko — self-hosted recipe management.

## Features
- Recipe CRUD with step-by-step editor
- Per-step ingredients with amount, unit, notes
- Photo upload
- Tags, work time, servings
- User authentication (JWT + Passkeys)
- Invite links for closed registration
- Admin panel for user management
- Public recipe gallery
- Share links (per-recipe and all-user)
- kitshn-compatible API compatibility layer (kitshn app support)
- EN + DE translations
- Docker Compose deployment
- 200+ seed ingredients, 30+ units (EN + DE)
- Light/dark mode

## Setup
See [README](README.md) for Docker Compose quick start." \
  --latest
```

**Step 5: Verify Docker images are built by CI**

Check Actions tab on GitHub for docker-publish workflow.

---

## Notes

- All commits: `git -c commit.gpgsign=false commit`
- Test database: use `DATABASE_URL` env var pointing to a local/test PostgreSQL
- If usage limits are hit: update the remote trigger `trig_013LaAaK3cVp8LSLCyJTJpxw` with a new cron 5h from now
- After each major task, consider using `superpowers:requesting-code-review` skill for self-review
- Use git worktrees for independent parallel work: `git worktree add .worktrees/<branch> -b <branch>`
