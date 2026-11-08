# Wokko

Your recipes, on your own server. A self-hosted recipe manager built to be simple to run and hard to lose data with. It works on phones and desktops, and with the [kitshn](https://github.com/aimok04/kitshn) mobile app.

[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](LICENSE)

## Features

- **Recipes:** steps with their own ingredients, photos, tags, serving scaling, search and cook log
- **Import:** from a link to most recipe sites and blogs, from pasted text, or from Tandoor, Mealie and Paprika exports, with a chance to check before saving
- **Offline:** the shopping list, this week's plan and recent recipes keep working without a connection, on HTTPS
- **Meal plans and shopping:** plan the week, add it to a shared shopping list, and sort that list by aisle for each shop
- **Cooking mode and printing:** one step at a time in big text, with timers and tick-off ingredients, also on shared links with no account. Any recipe prints cleanly
- **Recipe books and nutrition:** group recipes, and see estimates per serving from public-domain USDA data
- **kitshn:** use the kitshn mobile app, including creating and editing
- **AI assistants, if you want them:** connect Claude, ChatGPT or another MCP app. Wokko itself never calls an AI
- **Passkeys:** log in with a fingerprint or device PIN, on HTTPS or localhost
- **Your data is safe:** automatic verified backups, trash, version history and all-or-nothing saves
- **Any language:** English and German included, and adding another takes one file
- **Households and sharing:** share with a household, with friends, by link or publicly
- **Private by design:** people can't be searched, only reached by their exact handle
- **Easy to run:** one container and two folders, with no database server and no secrets to manage
- **Light and dark:** follows your system, or choose one yourself

## Installation

You need [Docker](https://docs.docker.com/engine/install/) with the Compose plugin, on Linux x86-64 or ARM64, such as a Raspberry Pi. There is no database to set up and no secret to create.

### With the ready-made image

This is the simplest way, and you don't need the source code.

```bash
mkdir wokko && cd wokko
curl -o docker-compose.yml https://raw.githubusercontent.com/Serraniel/Wokko/main/docker-compose.example.yml
docker compose up -d
```

Open `http://localhost:8080`, or `http://<server address>:8080` from another device, and create your admin account. The first account becomes the admin. After that, people join through invite links from **Admin**.

[`docker-compose.example.yml`](docker-compose.example.yml) explains each setting, including backups on a NAS and HTTPS. Images are on the [GitHub Container Registry](https://github.com/Serraniel/Wokko/pkgs/container/wokko), tagged by version such as `0.2.0` or `0.2`. The example file uses a fixed version instead of `latest`, so an update only happens when you choose one.

### From source

```bash
git clone https://github.com/Serraniel/Wokko.git
cd Wokko
docker compose up -d --build
```

The first build takes several minutes. To change a setting, copy `.env.example` to `.env`.

### Updating

Change the version in your `docker-compose.yml` to the one you want from the [releases](https://github.com/Serraniel/Wokko/releases), which also list what changed, then run:

```bash
docker compose pull
docker compose up -d
```

From source, run `git pull` and `docker compose up -d --build`. Wokko makes a backup before changing an existing database and stops if that fails, so updates are safe to automate. Going back to an older version after a database change isn't possible by itself, because a database from a newer version is refused rather than risked. Restore the backup made before the update instead.

## Configuration

Everything is optional.

| Variable | Default | Purpose |
|---|---|---|
| `PUBLIC_URL` | (empty) | The address people use, such as `https://wokko.example.com`. Needed for full share links, secure cookies, passkeys and Claude's hosted connectors |
| `BACKUP_INTERVAL_HOURS` | `24` | How often to back up |
| `BACKUP_RETENTION_DAYS` | `30` | How long to keep backups. The newest 3 are always kept |
| `TRASH_RETENTION_DAYS` | `30` | How long deleted recipes stay in the trash |

With the ready-made image, set these under `environment:` in `docker-compose.yml`. From source, `.env` also takes `WOKKO_PORT` (default `8080`).

## Keeping your data safe

Wokko backs itself up while running, verifies each backup and shows the status under **Admin**. It warns you if backups stop or fail, if disk space runs low, or if the database check finds a problem.

**Keep the data on a local disk.** A database on a network share can get corrupted, so Wokko refuses to start there. Backups are what belong on your NAS. In `docker-compose.yml`, replace the `wokko-backups` volume with a path such as `/mnt/nas/wokko:/backups`, and make that folder writable for Wokko's user: `sudo chown 65532:65532 /mnt/nas/wokko`.

**Don't rely on one copy.** If backups stay in the default Docker volume, they share a disk with your data, so a disk failure takes both.

### Restoring a backup

Stop Wokko, then run the restore command. It checks the backup first and moves your current data aside instead of deleting it.

```bash
docker compose stop wokko
docker compose run --rm wokko restore /backups/wokko-backup-2026-11-07T03-00-00Z.tar.gz
docker compose start wokko
```

A backup is a normal `tar.gz` with a SQLite database and your photos. Each user can also download their own recipes under **Settings** as a ZIP with JSON and photos, readable without Wokko.

## HTTPS, offline use and sharing from your phone

Wokko works over plain HTTP, but browsers keep these for HTTPS: using Wokko offline, installing it and sharing to it from other apps, passkeys, and keeping the screen on while cooking. Use a reverse proxy with HTTPS if Wokko is reachable from outside your network too.

[`docker-compose.example.yml`](docker-compose.example.yml) includes [Caddy](https://caddyserver.com), which gets the certificate by itself once a domain points at your server. Remove its comment signs and set your domain there and in `PUBLIC_URL`.

With another reverse proxy, set `PUBLIC_URL`, pass the original `Host` header through, and send `X-Forwarded-Proto: https`.

**Offline:** the shopping list, this week's plan and recently used recipes stay available. Ticking, adding and removing shopping items wait and are sent when you're back. Logging out deletes what's stored on the device.

**Sharing:** add Wokko to your home screen from the browser's menu, and it appears when you share a page from another app. Over plain HTTP, use the bookmark under **Integrations** instead.

## Moving from another app

Under **Integrations** (in **More** on a phone, or **Settings**), choose an export from Tandoor, Mealie or Paprika, or schema.org recipe files such as Nextcloud Cookbook's. You see what's inside first. Recipes arrive as "Only me", duplicates are skipped and the whole import can be undone.

## Using kitshn

In kitshn, enter the address your phone can reach Wokko at, such as `http://192.168.1.20:8080`, and log in with your handle and password. `localhost` won't work there, because on a phone that means the phone itself. Written against kitshn 2.1.5. Logging in and reading recipes are tested with the Play Store app on a real phone, and creating and editing with automated tests built from kitshn's source.

Recipes, photos, search and filters, cook logs, meal plans, shopping lists, recipe books, nutrition and importing from a link all work, including creating and editing.

These Tandoor features aren't in Wokko: AI import, shopping lists split into several named lists, and sharing a book with one person (books are shared with your household). Food names can only be edited in the web app, where every language is visible at once.

## Connecting an AI assistant

Wokko has a [Model Context Protocol](https://modelcontextprotocol.io) server at `/mcp`. It's off until an admin turns it on under **Admin**. An assistant can never do more than the person it acts for. It can't find other people, make recipes public or delete anything for good, and its changes appear under **Integrations**.

Set it up under **Integrations**: choose your app, such as Claude, ChatGPT, Claude Code, Codex, Gemini CLI, Cursor, VS Code, OpenClaw or Hermes Agent, and Wokko shows its steps. For Claude, Cursor and VS Code, a button opens the app with everything filled in. Apps that connect by address need no token. For the others, create a token there, then connect:

| Client | How |
|---|---|
| An app that takes an address and a header, on the same network | Address `http://your-server:8080/mcp` and header `Authorization: Bearer <token>` |
| Claude Code | `claude mcp add --transport http wokko http://your-server:8080/mcp --header "Authorization: Bearer <token>"` |
| Claude Desktop, or anything that starts a local MCP server, on the same computer as Wokko | Use `docker exec -i wokko /wokko mcp-stdio --token <token>` as its command |
| The same, on another computer with Docker | Use `docker run -i --rm ghcr.io/serraniel/wokko:<version> mcp-stdio --url http://your-server:8080/mcp --token <token>` as its command |
| Claude, ChatGPT or another app with custom connectors, on the web or phone | Add a custom connector with `https://your-server/mcp`, and approve it when Wokko asks, choosing whether it may only read. No token needed, but Wokko must be reachable from the internet over HTTPS, with `PUBLIC_URL` set. Tested with Claude |

## Adding a language

Copy `frontend/src/lib/i18n/messages/en.json` to `<code>.json` and translate it. The language appears in the app automatically, and the tests check that it's complete. Ingredient and unit names can be translated in the app, under **Ingredients**. Anything without a translation falls back to English.

## Development

Needs Go 1.26, Node 24 and Docker. The frontend's dev server proxies API calls to the backend.

```bash
make dev                  # backend on :8080
cd frontend && npm i && npm run dev

make test                 # all tests and type checks
make e2e                  # browser flows on a fresh instance, needs Chromium
make generate             # after changing SQL queries or API types
```

See `CLAUDE.md` for conventions. Contributions are welcome: please use [Conventional Commits](https://www.conventionalcommits.org/) and open a PR against `main`.

### Releases

Versions follow [semantic versioning](https://semver.org), and [release-please](https://github.com/googleapis/release-please) works them out from the commit messages on `main`. It keeps a release PR open with the next version and changelog. Merging that PR tags the release and publishes the image for x86-64 and ARM64.

Each [release](https://github.com/Serraniel/Wokko/releases) also has software bills of materials in SPDX and CycloneDX format, for the source and for each image, along with the third-party licenses, the image digest and checksums. To check that an image was really built here, run `gh attestation verify oci://ghcr.io/serraniel/wokko:<version> --owner Serraniel`.

The changelog is in [`CHANGELOG.md`](CHANGELOG.md), and in the app under **Changelog** in the footer.

## License

[EUPL-1.2](LICENSE)
