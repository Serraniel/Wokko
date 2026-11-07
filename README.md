# Wokko

Your recipes, on your own server. A self-hosted recipe manager built to be simple to run and hard to lose data with. It works on phones and desktops, and with the [kitshn](https://github.com/aimok04/kitshn) mobile app.

[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](LICENSE)

## Features

- **Recipes:** steps with their own ingredients, photos, tags, serving scaling, search and cook log
- **Import:** from a link to most recipe sites and blogs, or from pasted text, with a draft to check before saving
- **Your data is safe:** automatic verified backups, trash, version history and all-or-nothing saves
- **Any language:** English and German included, and adding another takes one file
- **Households and sharing:** share with a household, with friends, by link or publicly
- **Private by design:** people can't be searched, only reached by their exact handle
- **Easy to run:** one container and two folders, with no database server and no secrets to manage
- **Light and dark:** follows your system, or choose one yourself

## Quick start

```bash
git clone https://github.com/Serraniel/Wokko.git
cd Wokko
docker compose up -d --build
```

Open `http://localhost:8080` and create your admin account. Use a reverse proxy with HTTPS if Wokko is reachable from outside your network, and set `PUBLIC_URL` so share links and secure cookies work.

## Configuration

Everything is optional. Copy `.env.example` to `.env` to change a value.

| Variable | Default | Purpose |
|---|---|---|
| `PUBLIC_URL` | (empty) | The address people use, such as `https://wokko.example.com` |
| `WOKKO_PORT` | `8080` | Port on the host |
| `WOKKO_VERSION` | `0.1.0` | Image version to run |
| `BACKUP_INTERVAL_HOURS` | `24` | How often to back up |
| `BACKUP_RETENTION_DAYS` | `30` | How long to keep backups. The newest 3 are always kept |
| `TRASH_RETENTION_DAYS` | `30` | How long deleted recipes stay in the trash |

## Keeping your data safe

Wokko backs itself up while running, verifies each backup and shows the status under **Admin**. It warns you if backups stop or fail, if disk space runs low, or if the database check finds a problem.

**Keep the data on a local disk.** A database on a network share can get corrupted, so Wokko refuses to start there. Backups are what belong on your NAS. In `docker-compose.yml`, replace the `wokko-backups` volume with a path such as `/mnt/nas/wokko:/backups`, and make that folder writable for user ID 65532.

**Don't rely on one copy.** If backups stay in the default Docker volume, they share a disk with your data, so a disk failure takes both.

**Updating** is safe to automate. Before changing an existing database, Wokko makes a backup and stops if that fails. A database from a newer version than the running one is refused rather than risked.

### Restoring a backup

Stop Wokko, then run the restore command. It checks the backup first and moves your current data aside instead of deleting it.

```bash
docker compose stop wokko
docker compose run --rm wokko restore /backups/wokko-backup-2026-11-07T03-00-00Z.tar.gz
docker compose start wokko
```

A backup is a normal `tar.gz` with a SQLite database and your photos. Each user can also download their own recipes under **Settings** as a ZIP with JSON and photos, readable without Wokko.

## Using kitshn

Enter your Wokko address and log in with your handle and password. Written against kitshn 2.1.5.

| Works | Not yet |
|---|---|
| Log in, browse, search and filter by tag | Creating or editing recipes from kitshn |
| Cook from a recipe, with photos and scaling | Meal plans and shopping lists (shown empty) |
| Log a cook with a rating | Other Tandoor search filters, such as by rating |

## Adding a language

Copy `frontend/src/lib/i18n/messages/en.json` to `<code>.json` and translate it. The language appears in the app automatically, and the tests check that it's complete. Ingredient and unit names can be translated in the app, under **Ingredients**. Anything without a translation falls back to English.

## Development

Needs Go, Node and Docker. The frontend's dev server proxies API calls to the backend.

```bash
make dev                  # backend on :8080
cd frontend && npm i && npm run dev

make test                 # all tests and type checks
make e2e                  # browser flows on a fresh instance, needs Chromium
make generate             # after changing SQL queries or API types
```

See `CLAUDE.md` for conventions. Contributions are welcome: please use [Conventional Commits](https://www.conventionalcommits.org/) and open a PR against `main`.

## License

[EUPL-1.2](LICENSE)
