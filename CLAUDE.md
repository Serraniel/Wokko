# Wokko — Claude Development Guidelines

## Project Overview
Wokko is a self-hosted recipe management app. Go backend API + SvelteKit frontend + PostgreSQL.
License: EUPL-1.2. Starting version: 0.1.0.

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
- Use `git -c commit.gpgsign=false commit` for ALL commits (no GPG signing ever)
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
- Use `git commit` without `-c commit.gpgsign=false`

## Changelog Categories (for GitHub PR labels and release notes)
Auth, Recipes, Ingredients, Units, Sharing, MCP, i18n, UI, Infra, Fix

## Architecture Decisions
- JWT access tokens (15min) + refresh tokens (httpOnly cookie, 7 days)
- Passkeys via WebAuthn (go-webauthn library)
- Photos stored on filesystem at UPLOAD_PATH env var (default: /data/uploads)
- kitshn-compatible REST API at /api/ for mobile app support
- Semantic versioning starting at 0.1.0
- Footer version auto-populated from VITE_APP_VERSION env var (set at build time from git tag)
- No Redis needed — JWT sessions are stateless
- PostgreSQL only — no SQLite (avoids corruption issues)

## Logo
- Primary: red fiery wok character (#C0392B + #E67E22 flame)
- Ghost character from Gemini drafts is the 404 mascot only
- See frontend/static/logo.svg
