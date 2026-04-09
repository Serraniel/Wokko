# Wokko — System Design Document

**Date:** 2026-04-10  
**Version target:** 0.1.0  
**License:** EUPL-1.2

---

## 1. Overview

Wokko is a self-hosted recipe management web application. It is designed as a fast, lightweight, and maintainable alternative to Tandoor Recipes. The name "Wokko" is a playful take on "Wok" — cute, friendly, and cooking-focused.

### Goals

- Fast, responsive UI on desktop, tablet, and mobile
- Simple enough for non-technical users
- Easy self-hosting via Docker Compose
- Tandoor API compatibility for kitshn mobile app
- No mandatory email — privacy-first user management
- Passkey (WebAuthn) support
- EN + DE localization (more languages later)
- Easy backup and restore via Docker volumes
- MCP (Model Context Protocol) as opt-in feature

### Non-Goals (v0.1.0)

- Native mobile apps (kitshn handles mobile)
- Complex meal planning / shopping lists (post-MVP)
- Nutritional information (post-MVP)

---

## 2. Architecture

### Service Split

```
┌─────────────────────────────────────────────┐
│  Docker Compose                              │
│                                              │
│  ┌──────────────┐    ┌──────────────────┐   │
│  │  frontend    │    │  backend         │   │
│  │  SvelteKit   │◄──►│  Go (Echo)       │   │
│  │  :3000       │    │  REST API :8080  │   │
│  └──────────────┘    └────────┬─────────┘   │
│                               │              │
│                    ┌──────────▼─────────┐   │
│                    │  PostgreSQL :5432  │   │
│                    └────────────────────┘   │
│                                              │
│  Volumes: db-data, uploads                   │
└─────────────────────────────────────────────┘
```

### Deployment via Reverse Proxy

Users place Nginx or Caddy in front. The `PUBLIC_URL` env var configures the base URL. Both frontend and backend can be served under one domain:

- `/` → frontend (SvelteKit)
- `/api/` → backend (Go)

---

## 3. Backend (Go + Echo)

### Language & Framework

- **Go 1.22+** — single binary, low memory, fast, Pi-friendly
- **Echo v4** — lightweight, fast HTTP framework
- **sqlc** — type-safe SQL query generation from `.sql` files
- **golang-migrate** — database migrations
- **pgx v5** — PostgreSQL driver
- **go-webauthn** — passkey support
- **golang-jwt** — JWT session tokens

### Directory Structure

```
backend/
├── cmd/
│   └── server/          # main entrypoint
├── internal/
│   ├── api/             # HTTP handlers (grouped by feature)
│   ├── auth/            # JWT, passkey, invite logic
│   ├── config/          # env config
│   ├── db/              # sqlc generated code + migrations
│   ├── models/          # domain models
│   ├── services/        # business logic
│   └── storage/         # file upload handling
├── migrations/          # SQL migration files
├── sqlc.yaml
├── Dockerfile
└── go.mod
```

### Database: PostgreSQL

PostgreSQL chosen for robustness, ACID compliance, and easy backup (`pg_dump`). No SQLite to avoid corruption issues.

### Key Tables

- `users` — id, username, email (nullable), display_name, role, passkey_credential, created_at
- `invite_links` — id, token, created_by, used_by, expires_at, max_uses
- `recipes` — id, author_id, name, description, photo_path, tags[], work_time_min, is_public, created_at, updated_at
- `recipe_steps` — id, recipe_id, order, description, duration_min
- `step_ingredients` — id, step_id, ingredient_id, unit_id, amount, notes
- `ingredients` — id, name_en, name_de, default_unit_id, aliases[]
- `units` — id, name_en, name_de, abbreviation_en, abbreviation_de, aliases[]
- `ingredient_unit_stats` — ingredient_id, unit_id, use_count (for smart suggestions)
- `share_links` — id, recipe_id (nullable=all), token, expires_at
- `tags` — id, name (global, reusable)
- `mcp_tokens` — id, user_id, token_hash, scopes, created_at

### Authentication

1. **Username + password** (bcrypt)
2. **Passkeys** (WebAuthn via go-webauthn)
3. **JWT** access tokens (short-lived, 15min) + refresh tokens (7 days, stored in httpOnly cookie)
4. **Invite links** — admin generates tokens, user registers via `/register?invite=TOKEN`

Registration does NOT require email. Email is optional for password reset only.

### User Roles

- `admin` — full access, user management, invite links, system settings
- `user` — manage own recipes, view public/shared recipes

### Settings (stored in DB + env overrides)

- `OPEN_REGISTRATION` — bool, default `false`
- `ALLOW_PUBLIC_RECIPES` — bool, default `true`
- `MCP_ENABLED` — bool, default `false`
- `PUBLIC_URL` — base URL for share links
- `DATABASE_URL` — PostgreSQL connection string
- `UPLOAD_PATH` — file system path for uploads (default `/data/uploads`)

### Tandoor API Compatibility

A subset of the Tandoor v1 REST API will be implemented at `/api/` to ensure kitshn compatibility:

- `GET /api/recipe/` — list recipes
- `GET /api/recipe/{id}/` — recipe detail
- `GET /api/food/` — ingredients
- `GET /api/unit/` — units
- `GET /api/keyword/` — tags
- `POST /api/recipe/` — create recipe
- Auth: token-based (same format as Tandoor)

### MCP Protocol (opt-in)

When `MCP_ENABLED=true`, expose:
- `GET /mcp/resources/recipes` — list recipes
- `GET /mcp/resources/recipes/{id}` — recipe detail
- `GET /mcp/resources/ingredients` — list ingredients
- `GET /mcp/resources/units` — list units
- `POST /mcp/tools/create_recipe` — create recipe
- `PATCH /mcp/tools/update_recipe/{id}` — update recipe

OAuth 2.0 required for MCP access. Admin configures allowed clients in settings.

### File Storage

Recipe photos stored on local filesystem at `UPLOAD_PATH`. Files served via `/api/media/{filename}`. In Docker, this maps to a named volume `wokko-uploads`.

---

## 4. Frontend (SvelteKit + TailwindCSS)

### Stack

- **SvelteKit** — fast, small bundles, SSR/SSG capable
- **TailwindCSS v4** — utility-first styling
- **Paraglide-JS** — compile-time i18n for EN/DE (zero runtime overhead)
- **shadcn-svelte** — accessible component primitives

### Directory Structure

```
frontend/
├── src/
│   ├── lib/
│   │   ├── components/   # shared UI components
│   │   ├── api/          # typed API client (generated from OpenAPI)
│   │   └── stores/       # Svelte stores (auth, settings)
│   ├── routes/           # SvelteKit file-based routing
│   │   ├── (app)/        # authenticated routes
│   │   │   ├── recipes/
│   │   │   ├── ingredients/
│   │   │   ├── units/
│   │   │   └── admin/
│   │   ├── (public)/     # public/shared recipe views
│   │   ├── auth/         # login, register, passkey
│   │   └── setup/        # first-run wizard
│   ├── messages/         # Paraglide translation files
│   │   ├── en.json
│   │   └── de.json
│   └── app.html
├── Dockerfile
└── package.json
```

### UI Features

- Light/dark mode via TailwindCSS (system default + manual toggle)
- Mobile-first responsive design
- Recipe editor: step-by-step with per-step ingredient editor
- Ingredient/unit autocomplete with fuzzy search + "create new" option
- First-run wizard: admin account creation
- Admin panel: user management, invite links, settings toggle
- Public recipe gallery: no login required (if enabled)
- Share link generation per recipe or all recipes

### Footer

```
 GitHub  ·  v0.1.0  ·  Changelog  ·  EUPL-1.2  ·  Third-Party Notices
```

- Version fetched from GitHub releases API at build time (injected as env var)
- Changelog: categorized by PR labels (Auth, Recipes, Ingredients, Units, Sharing, MCP, i18n, UI, Infra, Fix)
- Third-Party Notices: generated at build time from `go-licenses` + `license-checker`

---

## 5. Data — Default Sets

To provide a useful out-of-the-box experience (similar to Tandoor's supermarket categories and food database):

- **Seed file:** `backend/migrations/seeds/` with SQL inserts
- **~200 common ingredients** in EN + DE with common units pre-assigned
- **~30 common units** in EN + DE (g, kg, ml, l, tsp, tbsp, cup, piece, pinch, etc.) with singular/plural forms
- **~20 common tags** (Vegetarian, Vegan, Gluten-free, etc.)
- Seeds run on first startup only (via a `seeded` flag in DB)

---

## 6. CI/CD

### GitHub Actions Workflows

```
.github/
├── workflows/
│   ├── pr-check.yml         # on PR: build + lint + test
│   ├── nightly.yml          # daily at 02:00 UTC: release if changes
│   └── docker-publish.yml   # on release tag: build + push to ghcr.io
└── dependabot.yml           # daily updates for actions, docker, go, npm
```

### Nightly Release Logic

1. Check commits since last release tag
2. If changes exist, bump patch version, create tag + GitHub release
3. Generate changelog from PR titles grouped by label
4. Trigger docker-publish workflow

### Docker Image

- Multi-stage builds (Go + Node separate stages)
- Final images based on `gcr.io/distroless/static` (Go) and `node:alpine` (SvelteKit)
- Published to `ghcr.io/serraniel/wokko`
- Tags: `latest`, `0.1.0`, `0.1`, `0`

### Semantic Versioning

- Major: breaking changes
- Minor: new features
- Patch: bug fixes, nightly bumps
- Start: `0.1.0`

---

## 7. Logo

Based on the Gemini drafts in `tmp/`:

- **Primary logo:** The red fiery wok character (cute, simple, fierce eyes, flames) — simplified for use as app icon
- **Favicon:** SVG version of the wok head/body silhouette with flame
- Color palette: deep red `#C0392B`, orange flame `#E67E22`, warm cream background `#FFF8F0`

The ghost character will not be used as the primary logo but may appear as an easter egg / 404 mascot.

---

## 8. Backup & Restore

Via Docker:

```bash
# Backup
docker exec wokko-db pg_dump -U wokko wokko > backup.sql
docker run --rm -v wokko-uploads:/data alpine tar czf - /data > uploads.tar.gz

# Restore
cat backup.sql | docker exec -i wokko-db psql -U wokko wokko
cat uploads.tar.gz | docker run --rm -i -v wokko-uploads:/data alpine tar xzf - -C /
```

The `docker-compose.yml` example will include all named volumes documented clearly.

---

## 9. GitHub Project Planning

### Milestones

1. **v0.1.0 — Foundation** — Repo setup, CI/CD, auth, basic recipe CRUD
2. **v0.2.0 — Recipe Features** — Steps, ingredients, units, photos, tags
3. **v0.3.0 — Sharing** — Public recipes, share links, public gallery
4. **v0.4.0 — kitshn Compat** — Tandoor API compatibility layer
5. **v0.5.0 — Admin & UX** — Admin panel, user management, invite links
6. **v0.6.0 — MCP** — MCP protocol, OAuth
7. **v0.7.0 — i18n** — Full EN/DE, seed data localization
8. **v1.0.0 — Stable** — All features complete, docs, performance audit
