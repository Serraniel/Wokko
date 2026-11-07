# Wokko — Claude Development Guidelines

## Project Overview
Wokko is a self-hosted recipe manager, an alternative to Tandoor. The owner's Tandoor database broke repeatedly, so **stability and data safety come first**. Go backend + Svelte frontend, embedded into one binary, with SQLite.
License: EUPL-1.2. Requirements and their reasoning: `docs/plans/2026-11-07-rewrite-requirements.md`.

## Repository Structure
- `backend/cmd/wokko` — entry point, plus the `restore` command
- `backend/internal/app` — business logic and API types. All permission and data rules live here
- `backend/internal/api` — HTTP layer of the web app, and all API tests
- `backend/internal/kitshn` — Tandoor-compatible API for the kitshn mobile app
- `backend/internal/store` — SQLite, migrations, sqlc queries
- `backend/internal/backup`, `media`, `auth`, `text`, `config`, `web`
- `frontend/` — SvelteKit static app. `src/lib/i18n/messages/*.json` holds the UI text
- `docs/` — Design docs and plans
- `.github/` — CI/CD workflows and Dependabot config

## Commit Convention
Use Conventional Commits with these scopes:
- `feat(auth)` — Authentication, user management, invites, friends, households
- `feat(recipes)` — Recipe CRUD, steps, photos, trash, versions
- `feat(ingredients)` — Ingredient management and suggestions
- `feat(units)` — Unit management
- `feat(sharing)` — Public recipes, share links
- `feat(kitshn)` — kitshn compatibility
- `feat(backup)` — Backups, restore, data safety
- `feat(mcp)` — Model Context Protocol integration
- `feat(i18n)` — Internationalization
- `feat(ui)` — Frontend UI components
- `fix(*)` — Bug fixes (use relevant scope)
- `chore(infra)` — Docker, CI, dependencies
- `chore(ci)` — GitHub Actions workflows

## ALWAYS
- Use `git -c commit.gpgsign=false commit` for ALL commits (no GPG signing ever)
- Run tests before committing: `make test`
- Use sqlc for all database queries — never raw SQL in application code. Only `store` runs SQL directly, for migrations and maintenance
- Run `make generate` after changing queries or `app/types.go`
- Save each user action in one transaction through `Store.Tx`
- Put new permission rules in `app`, so the web API and kitshn share them
- All API endpoints must have tests in `backend/internal/api/*_test.go`, including kitshn's
- Put every UI string and error code in the message files, in English first
- Style through the color tokens in `app.css`, so both themes keep working

## NEVER
- Add features beyond what is requested
- Skip database migrations — all schema changes via `backend/internal/store/migrations/`. Migrations only go forward
- Store secrets in code
- Skip error handling in Go — always handle errors explicitly
- Use `git commit` without `-c commit.gpgsign=false`
- Hardcode a language: German is one language among others, English is the fallback
- Let a user delete or catalog change remove recipe content implicitly
- Offer partial matching, listing or suggestions of other users. People are reached only by exact handle, friendship or household, on every API including kitshn's

## Changelog Categories (for GitHub PR labels and release notes)
Auth, Recipes, Ingredients, Units, Sharing, Kitshn, Backup, MCP, i18n, UI, Infra, Fix

## Architecture Decisions
- SQLite in WAL mode with `synchronous=FULL` and `foreign_keys=ON` on every connection, one writer connection. It replaced PostgreSQL on purpose: no separate service, credentials or major upgrades. Strict tables, stable integer IDs that are never reused
- The data directory must be on a local disk. The app refuses network filesystems. Backups go to the NAS
- One binary serves the API and the embedded frontend (`adapter-static`, no SSR)
- Opaque session tokens, stored hashed, in an httpOnly `sessionid` cookie. kitshn needs that exact name
- Passwords use argon2id. No required email: admins issue reset links
- Names of foods, units and keywords live in per-locale rows, so a new language needs no schema change
- API types are defined in `app/types.go`. Frontend types are generated from them
- Photos are re-encoded on upload and stored under `DATA_DIR/uploads`, and served only to people who may see the recipe
- kitshn compatibility targets a stated version (`kitshn.SupportedKitshn`). Its response types match kitshn's data classes

## Logo
- Primary: red fiery wok character (#C0392B + #E67E22 flame)
- The ghost is the 404 mascot only
- See frontend/static/logo.svg
