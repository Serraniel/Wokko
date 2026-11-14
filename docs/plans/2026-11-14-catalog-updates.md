# Starter catalog updates

Status: plan, not started. Tracking issue: #107.

## Problem

The starter catalog is 163 foods, 31 units and 25 tags in English and German, with USDA nutrition values. It is written by migrations `0002_seed.sql` and `0005_nutrition_seed.sql` exactly once, when a database is created.

- Fixes to an existing entry never reach existing installs, whether a typo, a better translation or a corrected nutrient value.
- A new language's names don't reach them either.
- New entries can only arrive through another one-off migration.

People do edit the catalog, so updates must never overwrite their changes, and must never remove anything a recipe uses.

## Idea: a shipped catalog pack, with provenance

**1. The catalog becomes data in the binary, not SQL.** It moves to `backend/internal/catalog/data/`, as JSON files embedded with `go:embed`, the same way as the kitchen knowledge pack.
- Every entry has a stable key such as `food.onion` or `unit.tablespoon`.
- Each entry holds its names per language (name, plural, abbreviation), aliases, default unit, nutrients and, later, food classes for the knowledge pack.
- The existing seed rows map to keys by their fixed IDs, so nothing is created twice.

**2. Wokko remembers what it shipped.** A new table records, for each catalog entry it ever added, the key, the entry's ID in this database, and a hash of the entry exactly as it was applied.

This shows whether someone changed an entry since. If the current entry still hashes to what was applied, nobody touched it.

**3. A sync on start-up, inside one transaction, after the usual backup before migrations:**

| Situation | What happens |
|---|---|
| A new entry in the pack, and no food or unit with that name exists | Added automatically |
| A new entry, but a person's own food already has that name | Not added. It is suggested to the admin as "link to the starter entry", which merges keeping the person's names as aliases |
| A changed entry that nobody edited | Updated automatically: names, aliases, nutrients |
| A changed entry that someone edited | Left alone. The admin sees the change as a suggestion, compared field by field, with "take the update" or "keep ours" |
| A name in a new language | Added if that language has no name yet. It never replaces one |
| Removed from the pack | Never deleted. The entry just stops getting updates |
| Merged or deleted locally | Remembered. The sync never brings it back |

Accepting or dismissing a suggestion is remembered by the pack version, so the same suggestion doesn't come back.

**4. The admin sees** under **Admin → Catalog updates**:
- what the last update added and changed, with the release it came from
- the open suggestions

No one has to do anything for updates that don't conflict.

This also lets kitshn and every other client see the improved names, since they read the same catalog.

## Keeping the pack up to date: a workflow

A scheduled GitHub Actions workflow, `catalog-refresh.yml`, runs monthly and can also be run by hand. It:

1. Runs a generator in `tools/catalog/`, which rebuilds the pack from its sources:
   - **USDA FoodData Central** for nutrients, pinned to a release. It's public domain.
   - **Wikidata** for translations into further languages, and for aliases and food classes like allergens. It's CC0.
   - Open Food Facts is ODbL, whose share-alike terms would apply to the shipped data. It is left out unless that's decided on purpose.
2. Validates the result:
   - every key is still present
   - no name is empty
   - nutrient values are plausible, for example under 100 g per 100 g and no more than 900 kcal
   - a change of more than 25 % to a nutrient is listed for review
3. If anything changed, opens or updates **one pull request**, "chore(catalog): refresh starter catalog from sources", with a readable summary: added entries, changed names and nutrient changes, each with its source.
4. A person reviews and merges it. The change ships with the next release. The commit type is `feat(ingredients)` for new entries and `fix(ingredients)` for corrections, so the changelog says what happened.

Hand-made improvements go into the same files through normal pull requests. These include German names that read better than Wikidata's, or entries asked for in issues. The generator keeps fields marked `"curated": true` instead of overwriting them.

**One catch for the bot PR:**
- Pull requests opened with the default `GITHUB_TOKEN` don't start CI.
- The release PR currently also needs a manual approval.

Both are solved by a small GitHub App, or a fine-grained token with only `contents` and `pull-requests` access on this repository, stored as a repository secret. Release-please can use the same one.

## Order of work

1. Move the seed into the pack format, with keys mapped to today's IDs. Add the provenance table, filled from the existing seed. Nothing changes for existing installs.
2. The start-up sync for unedited entries and new entries. Tests cover each row of the table above, including merged, deleted and edited entries.
3. Suggestions for edited entries, and the admin page.
4. The generator and the monthly workflow, with the bot credentials.
5. More languages from Wikidata, and then food classes for the kitchen knowledge pack.

## Not in scope

Fetching catalog data at runtime, deleting anything in existing databases, and changing people's own foods without the admin agreeing.
