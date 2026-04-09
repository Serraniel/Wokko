# Wokko

> Self-hosted recipe management — fast, lightweight, and yours.

Wokko is a modern alternative to Tandoor Recipes. It's designed to be quick, easy to use for non-technical users, and robust enough to run on a Raspberry Pi. Compatible with the [kitshn](https://github.com/aimok04/kitshn) mobile app.

[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](LICENSE)
[![Version](https://img.shields.io/github/v/release/Serraniel/Wokko)](https://github.com/Serraniel/Wokko/releases)

---

## Features

- **Recipe editor** — step-by-step cooking process with per-step ingredients
- **Smart ingredient suggestions** — fuzzy search with auto-suggested units based on usage history
- **Photo uploads** — store recipe photos locally in a Docker volume
- **Tags, work time, servings** — all the metadata you need
- **Passkey support** — passwordless login via WebAuthn
- **Invite-based registration** — closed by default, admin generates invite links
- **Public gallery** — share your recipes without requiring visitor accounts
- **Share links** — per-recipe or all-recipes shareable URLs
- **kitshn compatible** — implements the Tandoor v1 API for kitshn mobile app
- **EN + DE** — English and German UI out of the box
- **Light & dark mode** — system default + manual toggle
- **MCP protocol** — opt-in AI integration (OAuth-gated)
- **Docker-first** — easy self-hosting with named volumes for backup

---

## Quick Start

```bash
git clone https://github.com/Serraniel/Wokko.git
cd Wokko
cp .env.example .env
# Edit .env: set POSTGRES_PASSWORD, JWT_SECRET, PUBLIC_URL
docker compose up -d
```

Open `http://localhost:3000` — you'll be guided through the setup wizard to create your admin account.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_PASSWORD` | Yes | — | PostgreSQL password |
| `JWT_SECRET` | Yes | — | Secret for JWT signing (run: `openssl rand -hex 32`) |
| `PUBLIC_URL` | No | `http://localhost:3000` | Base URL for share links |
| `RP_ID` | No | `localhost` | WebAuthn relying party domain |
| `RP_ORIGIN` | No | `http://localhost:3000` | WebAuthn origin |
| `RP_DISPLAY_NAME` | No | `Wokko` | App name shown in passkey dialogs |
| `OPEN_REGISTRATION` | No | `false` | Allow public registration without invite |
| `ALLOW_PUBLIC_RECIPES` | No | `true` | Allow recipes to be made public |
| `MCP_ENABLED` | No | `false` | Enable MCP protocol endpoints |
| `WOKKO_VERSION` | No | `latest` | Docker image version tag |

---

## Backup & Restore

All data lives in two Docker named volumes:

```bash
# Backup
docker exec wokko-db pg_dump -U wokko wokko > backup.sql
docker run --rm -v wokko-uploads:/data alpine tar czf - /data > uploads.tar.gz

# Restore
cat backup.sql | docker exec -i wokko-db psql -U wokko wokko
cat uploads.tar.gz | docker run --rm -i -v wokko-uploads:/data alpine tar xzf - -C /
```

---

## kitshn Compatibility

Wokko implements a subset of the [Tandoor Recipes](https://github.com/TandoorRecipes/recipes) v1 REST API, making it compatible with the [kitshn](https://github.com/aimok04/kitshn) iOS/Android app.

Point kitshn at your Wokko instance URL and log in with your Wokko credentials.

---

## Nginx Reverse Proxy Example

```nginx
server {
    server_name wokko.yourdomain.com;

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        client_max_body_size 50M;
    }

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
    }
}
```

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Follow [Conventional Commits](https://www.conventionalcommits.org/) — see `CLAUDE.md` for scopes
4. Open a PR against `main`

---

## License

[EUPL-1.2](LICENSE) — European Union Public Licence v1.2
