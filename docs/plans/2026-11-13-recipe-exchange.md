# Recipe exchange: export and import between systems

Owner's request (2026-11-13): import and export of single recipes, a user's recipes, or all
recipes, to exchange them between systems.

## Where things stood

- **Settings > Integrations > Export** wrote a ZIP (`recipes.json` in format `wokko-export-1`,
  plus `photos/`) with the user's own recipes.
- **Import from a file** read Tandoor, Mealie, Paprika and schema.org files, but **not Wokko's own
  export**. Two Wokko servers couldn't exchange recipes at all.
- There was no export of a single recipe, and no export another app could import.

## Formats

| Format | What it's for | What survives |
|---|---|---|
| **Wokko** (`wokko-export-1`, ZIP) | Another Wokko | Everything a recipe has: steps with their own ingredients, amounts, units, notes, times, servings, tags, source and photo |
| **schema.org** (ZIP, one folder per recipe: `recipe.json` and `full.jpg`) | Other apps: Tandoor, Mealie, Nextcloud Cookbook and anything that reads schema.org recipes | Name, description, servings, times, tags, source, photo, steps, and ingredients as one list of lines |

The schema.org layout is Nextcloud Cookbook's, which Tandoor and Mealie both import. Wokko reads it
back too.

## Scopes

| Scope | Who | Where |
|---|---|---|
| One recipe | Anyone who can see it, including through a share link or the public gallery | Recipe page menu, and shared or public recipe pages |
| My recipes | The signed-in user | Integrations |
| All recipes I can see | The signed-in user: their own, their household's, friends' and recipes shared with them | Integrations |

**"All recipes" doesn't mean every recipe on the server.** An admin export of other people's
private recipes would break "private by design": a stolen admin session would expose everyone. A
whole server moves with a backup and `wokko restore`, which keeps owners, sharing and history.

## Import rules (unchanged)

Imported recipes belong to the importer, start as "Only me", and duplicates by name are skipped.
The importer sees what's inside first and can undo the whole import.

## Issues

1. Import Wokko's own export files, losslessly.
2. Export a single recipe, as Wokko or schema.org, also from share links and the public gallery.
3. Export my recipes or all recipes I can see, as Wokko or schema.org.
