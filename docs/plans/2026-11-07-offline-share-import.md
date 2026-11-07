# Offline, share to Wokko, and importing other apps' files

The owner chose these from `2026-11-07-ideas.md`.

## 1. Import from other apps

A file import on a new page, `/import`, for:

| Source | File | Notes |
|---|---|---|
| Tandoor | The default export: a zip of zips, each with `recipe.json` and an image | Keeps steps with their ingredients |
| Mealie | A zip with `recipes/<slug>/<slug>.json` and `images/original.*`, old and new field names | |
| Paprika | `.paprikarecipes`: a zip of gzipped JSON, with the photo inside | Plain text ingredients, read by the existing parser |
| Anything with schema.org JSON | Single files, or a zip such as Nextcloud Cookbook's | |

- **Check first.** The same upload with `dry_run` lists what's in the file and what looks like a duplicate. Nothing is saved.
- **One recipe, one transaction.** A broken recipe is reported and the rest still arrive. Everything is private at first, and duplicates are skipped, so importing twice is harmless.
- **Undo.** The result page can move the whole import to the trash.
- **Hostile files.** Limits on upload size, entry size, total unpacked size, entry count and nesting depth. Nothing is written to disk from inside an archive, and photos go through the normal upload path, which re-encodes them.

## 2. Offline

A service worker, which browsers only allow on **HTTPS or localhost**.

- **App code:** precached by version. Pages try the network first, so an update is never held back by an old copy.
- **Data:** recipes, plans, shopping and books try the network first and fall back to the last answer. Only an explicit list of read-only addresses is stored, never admin data, tokens or anything from kitshn or MCP.
- **Shopping offline:** ticking, adding and removing go into a queue that's sent in order when the connection returns. Other changes need a connection, and the editor's drafts already protect work in progress.
- **Privacy:** logging out, or a session ending, deletes the stored data.

Stale code is the known risk of service workers, and the reason the overhaul left this out. Precached files are versioned, old versions are removed on activation, and a navigation always prefers the network.

## 3. Share to Wokko

- **Phone:** the installed app registers as a share target, which needs HTTPS. Sharing a page opens the import with that link and starts it. Plain text goes to the paste box.
- **Any browser, any address:** a bookmark that sends the current page to the import. It also works over plain HTTP.

## Testing

Go tests with files built to the formats in Tandoor's own importers (`cookbook/integration`), including corrupt and oversized files. Browser flows go offline for real and come back.
