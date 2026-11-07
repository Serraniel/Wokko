# Wokko Rewrite: Requirements Draft

**Date:** 2026-11-07
**Status:** Agreed and in implementation on the `rewrite` branch. Section 8 shows what's done.

**Sources:** GitHub issues #1–#50, PRs, `README.md`, `CLAUDE.md`, `docs/plans/*`, the code on `feat/language-preference` (main plus 9 commits, PR #57), the screenshots in `tmp/`, the brief for this rewrite and the owner's answers, the owner's Tandoor compose file, a test restore of the newest Tandoor backup, and the kitshn v2.1.5 API client source.

**Provenance tags**

| Tag | Meaning |
|---|---|
| `[U]` | Stated by the owner in the rewrite brief or answers |
| `[#n]` | GitHub issue |
| `[D]` | Design doc or README |
| `[P]` | **Proposed in this draft.** Not previously requested, needs a yes or no |

**Status in the old code**

| Status | Meaning |
|---|---|
| ✅ | Works, from reading the code |
| 🟡 | Partly there or broken |
| ❌ | Missing |

**How this was checked:** `go build`, `go vet` and the `internal/auth` tests pass locally. The app was not run, and the API integration tests were not run because they need a database. Behaviour claims come from reading the code, unless noted otherwise.

---

## 1. Why this project exists

Wokko is a self-hosted alternative to Tandoor. `[U]` The owner's Tandoor database was destroyed several times on a local setup. `[U]`

The old project's other goals were: fast, simple enough for non-technical users, easy to host with Docker, able to run on a Raspberry Pi, no mandatory email, EN and DE. `[D]`

### What we know about the Tandoor failures

**The root cause can't be determined.** No logs from the failures survive. The owner recalls setting it up twice and the database container "just stopped working" both times, possibly also once on a VPS. The current compose file was set up after the last failure, so it may not match the setup that broke.

**Evidence from the newest backup (2026-11-06, restored into an isolated test container):**

| Finding | Detail |
|---|---|
| The backup is healthy | Postgres started from it after a clean shutdown. A full `pg_dump` succeeded and `pg_amcheck` reported nothing |
| This database has survived since 2026-02-21 | Its first migrations ran that day and it has stayed in use since. There are no signs of a rebuild |
| Very little data is at stake | 1 user, 2 recipes, 26 ingredient rows, 393 foods, 1,488 food properties, no meal plans or shopping entries |
| Tandoor updates itself unattended | Its migrations ran at 03:28, 01:14, 01:07 and 02:25 UTC on four separate nights, and the image is untagged (`latest`) |
| Backups silently stopped for 12 days | Nothing exists for 2026-10-09 to 10-20, inside the 30-day retention window. Nothing would have alerted anyone. The cause is unknown |
| File-level backups are large and stop the app | Each archive is about 58 MB, mostly Tandoor's static files. A `pg_dump` of the database was about 0.5 MB |

The older "maybe broken" backup was not found. A search of the home directory and `/mnt` timed out, so it may still exist.

**Plausible causes, none confirmed.** Each could look like "the database container just stopped":

| Hypothesis | Why it fits | Design response |
|---|---|---|
| Data on network storage (SMB/NFS) | The owner remembers possibly using the NAS. Postgres data on a network share is a known source of startup and corruption problems | DS-20 |
| An unattended image update, such as a new Postgres major version that can't read the old data directory | Unattended updates are proven for the Tandoor image. Whether the database image was ever affected is unknown | DS-13, DS-16, DS-23 |
| Credentials changed in `.env` | Postgres only reads `POSTGRES_USER` and `POSTGRES_PASSWORD` when first creating the database. The compose file also mixes env values with hardcoded `djangouser` and `djangodb` | DS-22 |
| Disk full or out of memory, especially on a small VPS | Common reason for a Postgres container to stop | DS-21, DS-25 |
| Failing disk or power loss | Can't be excluded | DS-8 to DS-11 |

**Conclusion:** "PostgreSQL, no SQLite" does not address any of these. Wokko has to assume the failure can recur, make it hard to cause, explain it clearly when it happens, and make recovery routine.

---

## 2. What the old project actually is

About 5,750 lines of Go (including generated sqlc code) and about 2,000 lines of frontend. The README and the issue tracker both describe it inaccurately:

- The README lists passkeys, kitshn compatibility, MCP and EN/DE UI as features. None of them work.
- Issues #1–#23 are all open, although some are implemented. They have one-line descriptions and no comments.
- #42, #46 and #48 are still open, although merged PRs claim to fix them.

### Defects that shape the requirements

| Finding | Where | Consequence |
|---|---|---|
| The app never opens a database transaction | No `Begin` in `backend/internal`; the only `WithTx` is unused generated code | A failed multi-step write leaves partial data |
| Saving a recipe takes many separate API calls | `RecipeEditor.svelte:108-133` | A failure midway leaves a half-saved recipe. Ingredient and photo errors are only logged to the console, so ingredients can vanish silently |
| Editing a recipe re-creates all its steps | `RecipeEditor.svelte:118` and `UNIQUE(recipe_id, step_order)` | Editing a recipe with steps should fail with a 500 after the metadata has already saved |
| A photo is deleted before its replacement is saved | `UploadRecipePhoto` in `recipe_handler.go` | A rejected or failed upload loses the old photo |
| Ingredient merge is two non-atomic writes | `MergeIngredients` in `ingredient_handler.go` | A failure between them leaves the merge half-done |
| Deleting a user cascades to their recipes | `recipes.author_id ... ON DELETE CASCADE` | One admin click permanently deletes that user's recipes |
| Database errors are often discarded | `_ =` and `tags, _ :=` throughout the handlers | Failures go unnoticed, and this breaks a `CLAUDE.md` rule |
| Any logged-in user can edit or merge shared ingredients and units | `apiGroup` in `routes.go`, no admin check | #19 says this is admin-only |
| Step update does not check the step belongs to the recipe | `UpdateRecipeStep` | The owner of any recipe can overwrite another user's step if they have its ID |
| Photos of private recipes are served without login | `e.Static("/api/media", ...)` | Anyone with the URL can see them |
| Disabled users can still log in | `Login` never checks `is_active` | The disable flag has no effect |
| The admin settings toggles do nothing | `Register` reads the env config, not the DB value; `AllowPublicRecipes` is never checked | The UI claims control it doesn't have |
| Uploads are checked by file extension only | `storage.go` | Any content can be stored under an allowed extension |
| No login rate limiting | `main.go`, `routes.go` | Passwords can be brute-forced |
| Tests cover few endpoints | API tests exist only for auth, recipes and ingredients. Admin, public, share and kitshn have none. The frontend has no tests | Breaks a `CLAUDE.md` rule |

### Recurring bug pattern

Many closed issues come from the architecture rather than from individual mistakes:

- **The frontend server sits between the browser and the API:** #32 (proxy), #45 (the server-side refresh rotated the browser's token).
- **Auth state is split between `localStorage` and a cookie:** #45, #47, #50, and the 401 refresh fix in PR #55.
- **Frontend and API shapes are defined separately:** #44, #48 (field name mismatches).

A rewrite should remove these causes (see section 6).

---

## 3. Data safety requirements

This is the reason the project exists, and the old project barely addressed it: its only measures were choosing PostgreSQL and a manual backup snippet in the README. **All of this section is `[P]`, derived from `[U]`.**

### Writes

| ID | Requirement |
|---|---|
| DS-1 | Each user action is one transaction. A recipe with its steps, ingredients and tags saves completely or not at all, through one API call |
| DS-2 | Unsaved editor input survives a failed save, an expired session or a closed tab (local draft) |
| DS-3 | Two people or tabs editing the same recipe get a conflict warning instead of silently overwriting each other |

### Deletion

| ID | Requirement |
|---|---|
| DS-4 | Deleted recipes go to a trash and can be restored for a set period |
| DS-5 | Deleting a user never deletes recipes implicitly. The admin chooses to reassign or delete them, with a clear warning |
| DS-6 | Recipes keep a version history, so a bad edit can be undone |
| DS-7 | Foreign keys protect recipe content from being removed by a cascade from shared data |

### Backup and restore

| ID | Requirement |
|---|---|
| DS-8 | Automatic scheduled backups are on by default, with retention, and include both the database and photos |
| DS-9 | Backups are stored outside the database volume and can be pointed at another disk or NAS |
| DS-10 | The admin UI shows last backup time and status, with a visible warning when backups are failing or stale. The 12-day Tandoor gap is the test case |
| DS-11 | Restore is documented, and CI tests it end to end. A backup that has never been restored doesn't count |
| DS-12 | Users can export everything in an open, documented format (JSON and images) that is usable without Wokko |
| DS-24 | Backups use `pg_dump` plus uploaded files. They do not stop the app, don't include rebuildable files, and can be restored into a newer Postgres version |

### Upgrades and storage

| ID | Requirement |
|---|---|
| DS-13 | The app takes a verified backup before applying migrations, and refuses to migrate if that fails |
| DS-14 | A failed migration leaves the data untouched and stops startup with a clear error |
| DS-15 | The app refuses to start against a database newer than it understands, which protects downgrades |
| DS-16 | There is a documented path for PostgreSQL major upgrades. A changed major image tag is the classic way a Docker database appears to "break" |
| DS-17 | Seed data is applied in a way that never overwrites user changes |
| DS-18 | Startup and scheduled integrity checks report DB and upload problems, such as photos missing from disk |
| DS-19 | Photos are written atomically: new file first, then the DB update, then old file cleanup |
| DS-20 | The default setup keeps the database on a local Docker volume, and the docs warn against network shares. A network filesystem is detected and warned about wherever possible. Backups are what go to the NAS |
| DS-21 | Common failures produce a plain explanation and the fix: wrong credentials, incompatible database version, disk full, database unreachable. The web app shows a maintenance page, not a blank error |
| DS-22 | Database credentials can't silently drift from the existing database. Setup uses one secret in one place, and a mismatch gives a clear error (DS-21) |
| DS-23 | Published compose files pin the Postgres major version and a Wokko version, never `latest`. Unattended updates must be safe because of DS-13, or the docs must advise against them |
| DS-25 | Admin UI warns about low disk space for the database, uploads and backups |

---

## 4. Functional requirements

Priority is a proposal: **M** = first usable release, **L** = later, **?** = needs a decision.

### Accounts and access

| ID | Requirement | Source | Old | Pri |
|---|---|---|---|---|
| A-1 | First-run setup creates the admin, and the app redirects to it until done | #3, #36 | ✅ | M |
| A-2 | Username and password login, with email optional | #2, D | ✅ | M |
| A-3 | Passkey registration and login | #2, D | ❌ 501 stubs, library not installed | L |
| A-4 | Invite links with expiry and usage limit | #17 | ✅ | M |
| A-5 | Invite-only by default, with an open registration setting that takes effect | #18, D | 🟡 | M |
| A-6 | Invite code field always visible, with clear errors | #34 | ✅ | M |
| A-7 | Admin can list, disable and delete users | #16 | 🟡 Disable missing | M |
| A-8 | Admin link and page work reliably | #47, #50 | ✅ | M |
| A-9 | Users can change their password | P | ❌ | M |
| A-10 | Password recovery without email, such as an admin-issued reset link | P (D planned email-only) | ❌ | M |
| A-11 | Login rate limiting | P | ❌ | M |
| A-12 | Several users on one instance, starting with one | U | 🟡 | M |
| A-13 | Households: users can belong to a household that shares recipes, meal plans and shopping lists | U | ❌ | M |
| A-14 | Share a recipe with a specific user | U | ❌ | M |
| A-15 | Users can't be discovered. There is no directory, search or autocomplete: typing "john" never suggests "John Doe" | U | n/a | M |
| A-16 | Share with a specific user only by entering their complete, exact handle, or by choosing from your own friends | U | n/a | M |
| A-17 | Friends: a mutual connection created from an exact handle or a friend link. Friends appear in your own share picker, and nobody can see anyone else's friends | U, P (mechanism) | ❌ | M |
| A-18 | Handle lookups are rate-limited. An exact handle can be confirmed, because a typo must not silently share nothing, but guessing at scale is blocked | P | n/a | M |
| A-19 | Public recipes and share links don't expose the author's handle, or any way to browse that author's other recipes unless they opted in | P | ❌ | M |

### Recipes

| ID | Requirement | Source | Old | Pri |
|---|---|---|---|---|
| R-1 | Create, read, update and delete recipes | #4 | 🟡 Editing is broken | M |
| R-2 | Fields: name, description, work time, servings, source URL | D, schema | 🟡 Source URL has no UI | M |
| R-3 | Ordered steps with instructions and optional duration | #6 | 🟡 | M |
| R-4 | Ingredients attached to each step, with amount, unit and note | #6 | 🟡 | M |
| R-5 | Combined ingredient list on the recipe page | Existing UI | ✅ | M |
| R-6 | Recipe photo upload, replace and delete | #7 | 🟡 Single photo, unsafe replace | M |
| R-7 | Tags, with filtering by tag | #10 | 🟡 No filtering | M |
| R-8 | Search on the server, not only within the loaded page | P (old search filters loaded recipes only) | 🟡 | M |
| R-9 | Scale ingredient amounts by servings | P | ❌ | M |
| R-10 | Reorder steps in the editor | P | ❌ | M |
| R-11 | Waiting time, separate from work time | kitshn requires it | ❌ | M |
| R-12 | Step-by-step cooking mode in the web app | P (cooking mainly happens in kitshn) | ❌ | L |
| R-13 | Import a recipe from a URL | P (kitshn calls `/recipe-from-source/`) | ❌ | L |
| R-14 | Bring over the 2 existing Tandoor recipes and possibly the food data as a one-off, not as a general importer | P | ❌ | M |
| R-16 | Ratings and cook log, including last-cooked date | kitshn | ❌ | M |
| R-17 | Recipe books | kitshn | ❌ | L |
| R-15 | Decimal or fraction display | P (seen in Tandoor's setup screen) | ❌ | L |

### Ingredients, units and seed data

| ID | Requirement | Source | Old | Pri |
|---|---|---|---|---|
| I-1 | Search and autocomplete for ingredients and units in the editor | #8, #9, D | 🟡 Substring match, not fuzzy | M |
| I-2 | Create an ingredient or unit inline while typing | #43 | ✅ Wired up in PR #56; #43 still open | M |
| I-3 | Unit suggested from usage history and default unit | README, D | 🟡 Schema exists, end-to-end use unverified | M |
| I-4 | Pages to browse and manage ingredients and units | #8, #9, #49 | ❌ API only, no UI | M |
| I-5 | Merge duplicates and manage aliases, admin only | #19 | 🟡 API has merge, no UI, no admin check | M |
| I-6 | Singular and plural forms | D, schema | 🟡 Schema only | L |
| I-7 | Seed data for EN and DE | #11, #23 | ✅ 163 ingredients, 31 units, 25 tags | M |
| I-8 | Show seed counts in admin, with an optional "import more" action | #49 | ❌ | L |
| I-9 | A larger seed set than 163. #49 notes Tandoor ships about 2,000 | #49 | ❌ | ? |

### Sharing

| ID | Requirement | Source | Old | Pri |
|---|---|---|---|---|
| S-1 | Mark a recipe public | D | ✅ | M |
| S-2 | Public gallery without login, with an instance switch that takes effect | #12, D | 🟡 Switch not enforced | M |
| S-3 | Public recipe page shows the full recipe | #12 | 🟡 API returns no steps or ingredients | M |
| S-4 | Per-recipe share links, with optional expiry and revocation | #13 | 🟡 Link goes to `/share/...`, which has no frontend page. No revoke | M |
| S-5 | Share link covering all of a user's recipes | #14 | ❌ Schema allows it, no endpoint | L |
| S-6 | Private recipe photos need authorization | P | ❌ | M |

### Language and UI

| ID | Requirement | Source | Old | Pri |
|---|---|---|---|---|
| U-1 | Full EN and DE UI | #22 | ❌ All strings are English, Paraglide not installed | M |
| U-2 | Ingredient and unit names follow the user's language | #23, PR #57 | 🟡 Unmerged | M |
| U-3 | Language setting saved per user | P (old setting was per browser) | ❌ | M |
| U-4 | Room to add languages without schema changes | D ("more languages later") | ❌ `name_en`/`name_de` columns | L |
| U-5 | Light and dark mode, following the system with a manual toggle | #41, #42, #46 | ✅ | M |
| U-6 | Mobile-first responsive layout | D | 🟡 Not audited | M |
| U-7 | Autofocus the first field on forms | #38 | ✅ | M |
| U-8 | Footer: GitHub, version, changelog, license, third-party notices | D | 🟡 Changelog and notices are hardcoded | L |
| U-9 | Red wok logo, ghost as 404 mascot | CLAUDE.md | 🟡 Logo present, no mascot page | L |
| U-10 | Mascot eyes follow the cursor on login and register | #40 (low priority) | ❌ | L |

### Integrations

| ID | Requirement | Source | Old | Pri |
|---|---|---|---|---|
| X-1 | kitshn mobile app works against Wokko. **Very important to the owner, the main way to cook** | U, #15, D | ❌ See section 5 | M |
| X-2 | Opt-in MCP server for AI tools | #20 | ❌ Only a table and a settings flag | L |
| X-3 | OAuth 2.0 for MCP | #21 | ❌ | L |

### Meal plans, shopping lists and nutrition

All three are now in scope. `[U]` The old design had deferred them. They come after the core, and each must also work through kitshn.

| ID | Requirement | Source | Old | Pri |
|---|---|---|---|---|
| P-1 | Meal plan calendar with recipes or notes, servings and date range | U, kitshn | ❌ | L |
| P-2 | Custom meal types, such as breakfast or dinner | kitshn | ❌ | L |
| P-3 | Meal plans can be shared within a household | U, kitshn (`shared`) | ❌ | L |
| L-1 | Shopping list with items, amounts, units and checked state | U, kitshn | ❌ | L |
| L-2 | Add a recipe or meal plan to the list, scaled by servings | kitshn | ❌ | L |
| L-3 | Lists shared within a household, updated across devices | U (household), P | ❌ | L |
| L-4 | Supermarkets and categories for sorting the list | kitshn | ❌ | L |
| F-1 | Nutrition properties per food, such as energy, fat and protein | U, kitshn | ❌ | L |
| F-2 | Nutrition calculated per recipe and serving | U, kitshn | ❌ | L |
| F-3 | Starting nutrition data, subject to a license check. The existing Tandoor data has 1,488 food properties, but its origin and license are unverified | P | ❌ | L |

### Delivery order (proposal)

1. **Foundation and kitshn proof:** data model, data safety, login, and browsing and cooking recipes in kitshn on your phone.
2. **Daily use:** web editor, ingredients, users, households, sharing, EN and DE.
3. **Planning:** meal plans and shopping lists, on web and kitshn.
4. **Extras:** nutrition, passkeys, MCP and the remaining L items.

---

## 5. kitshn compatibility: must-have and largest scope risk

The owner cooks with kitshn, so it shapes the whole design and can't be added on later.

### Why the old layer can't work

- It has no login flow kitshn uses.
- It returns CRC32 hashes of UUIDs as integer IDs, then parses `/recipe/:id/` as a UUID. **No ID from the list can be fetched.**
- CRC32 IDs can also collide.

### What kitshn v2.1.5 expects

This comes from reading its client source. **It has not been tested against a server yet.**

| Area | Expected behaviour |
|---|---|
| Connection test | `GET /api/` returns JSON. An unauthenticated request must return exactly 403 |
| Login | Imitates django-allauth: `GET /_allauth/browser/v1/auth/session` for a `csrftoken` cookie, then login returning `sessionid`. It also accepts a `Bearer` token. The exact path handling still needs confirming |
| Version | `GET /api/server-settings/current/` returns a `version` string, recorded by the client. Whether features depend on it still needs checking |
| IDs | **Integers everywhere** |
| Recipe fields | Includes keywords, steps, `working_time`, `waiting_time`, `servings`, `servings_text`, `internal`, `properties`, `food_properties`, `rating`, `last_cooked`, `new` and `recent` |
| Recipe search | Filters for keywords, rating, times cooked, creator, creation date, sort order, `new` and `random`, using paged responses |
| API areas | Recipes, steps, foods, units, keywords, cook log, meal plans, meal types, recipe books, shopping, supermarkets and categories, spaces, users, user preferences, server settings, recipe import and AI import |

Tandoor 2's API has many more endpoints than this. Wokko only needs what kitshn calls, which is large but finite.

### Risks

- **kitshn can change at any time:** it's built for Tandoor (currently 2.7.2), and an update could break Wokko. Mitigation: state the supported kitshn version, and run contract tests in CI against that version's models.
- **Auto-updating phone apps:** a kitshn update could reach the phone before Wokko supports it. Mitigation: track new kitshn releases and test them early.
- **Imitating another product's login:** Wokko has to look like Tandoor to kitshn. It may be worth asking the kitshn maintainer about explicit support for compatible servers.

### Requirements

| ID | Requirement |
|---|---|
| K-1 | The first milestone proves login, recipe list, search and recipe detail in the Play Store version of kitshn on the owner's phone, before other features are built |
| K-2 | Every entity kitshn sees has a stable integer ID stored in the database, never a hash |
| K-3 | The data model covers the concepts kitshn exposes: foods, keywords, properties, spaces, cook logs. A Wokko household maps to a Tandoor "space" |
| K-4 | kitshn compatibility is a separate layer using the same services and permissions as the web API, so private recipes stay private on both |
| K-5 | CI contract tests cover each kitshn endpoint Wokko implements, against a stated kitshn version |
| K-6 | Writes from kitshn have the same data safety as the web app, including transactions and trash |
| K-7 | Features Wokko doesn't implement, such as AI import, fail cleanly in kitshn. *Exact behaviour to be tested* |
| K-8 | kitshn's `GET /api/user/` returns only the caller, their household and their friends, never all users, so it can't bypass A-15 |

---

## 6. Non-functional requirements and architecture direction

These are starting points for discussion, not decisions. **Proposals are `[P]`.**

**Requirements**

| ID | Requirement | Source |
|---|---|---|
| N-1 | Runs comfortably on a Raspberry Pi | README |
| N-2 | Installs with `docker compose up`, needing only a few env variables | D |
| N-3 | EUPL-1.2 license, semantic versioning, Conventional Commits | CLAUDE.md |
| N-4 | Secrets come from the environment, and the app refuses to start with default secrets. The old compose file falls back to `changeme` | CLAUDE.md, P |
| N-5 | Every API endpoint has tests, and data safety paths have failure tests | CLAUDE.md, P |
| N-6 | Releases only when meaningful, instead of the old automatic nightly patch bump | P |

**Keep:** Go, sqlc, versioned migrations, Svelte, Tailwind, EUPL license, logo. They were not the source of the problems. Useful parts of the old project, such as the seed data, can be reused.

**Change**

| Proposal | Fixes |
|---|---|
| Serve a static Svelte app from the Go binary, giving one app container plus Postgres | The proxy and SSR bug class (#32, #45), plus lower memory use on a Pi |
| Use one httpOnly session cookie with server-side sessions instead of JWT in `localStorage` | The auth bug class (#45, #47, #50). Disabling a user takes effect immediately. **Changes a decision recorded in `CLAUDE.md`** |
| Generate TypeScript API types from one OpenAPI spec | The field mismatch bug class (#44, #48) |
| Save a recipe as one document through one transactional endpoint | The partial-save and edit defects, DS-1 |
| Use integer IDs as primary keys, with random tokens wherever an ID would be exposed publicly | K-2, without making public links guessable |
| Use a translations table for names | U-4 |
| Keep a separate backup service or job in the compose stack | DS-8 to DS-11 |

### Database decision

**Decided: SQLite in WAL mode, inside the Wokko app container.** The owner left the choice to this draft with stability as the goal. This reversed the old "PostgreSQL only" rule. A test confirmed the driver, transactions, foreign keys, concurrent writers and backup before work began.

The goal is stability. PostgreSQL is an excellent engine, and nothing suggests it corrupted anything by itself: the Tandoor backup is clean. The risk is everything around it, and running it as a separate service is what makes most of the section 1 hypotheses possible.

| Failure from section 1 | PostgreSQL service | SQLite inside the app |
|---|---|---|
| New major version can't read old data | Possible, needs a dump and restore or `pg_upgrade` | Doesn't apply: the file format is stable and the version ships with the app |
| Credentials drift from `.env` | Possible | Doesn't apply: there are no credentials |
| Database container stops or is unreachable | Possible | Doesn't apply: there is no separate service |
| Database on a network share | Risky, and the app can't see where the files are | **Even more dangerous**, but the app can see it directly and refuse to start (DS-20) |
| Disk full | Possible | Possible, with direct warnings (DS-25) |
| Power loss or failing disk | Backups needed | Backups needed. WAL with full sync protects against power loss, not disk failure |

**What SQLite costs**

- One writer at a time. That's fine for households and kitshn, but it's not suited to a large public instance.
- One app instance only, with no horizontal scaling.
- Fewer database features. Search uses a normalized text column, which also folds umlauts in any language and handles short words.
- A lost option: moving to PostgreSQL later would be a real migration. Keeping queries in sqlc makes that manageable.

**These requirements changed as a result**

| ID | Change |
|---|---|
| DS-8, DS-24 | Backups use SQLite's online backup, without stopping the app |
| DS-11 | CI still restores a backup and also runs `PRAGMA integrity_check` |
| DS-16, DS-22 | Removed, since these failure modes no longer exist |
| DS-20 | Strict: the app refuses to start with its database on a network filesystem. Backups still go to the NAS |
| DS-23 | Only the Wokko version needs pinning |
| N-2 | Installation becomes one container and two folders: data and backups |


---

## 7. Questions

### Answered

| # | Question | Answer | Effect |
|---|---|---|---|
| Q1 | What happened to the Tandoor database? | Unknown. It broke twice, and the container just stopped | Section 1, DS-20 to DS-25 |
| Q2 | Is there data to bring over? | A healthy backup exists with 2 recipes. An older backup may be broken | R-14 as a one-off. No migration from the old Wokko schema |
| Q3 | Who uses it? | Just the owner now, but multi-user with public, per-user and household sharing | A-12 to A-15 |
| Q4 | Which other features? | Nutrition, meal plans and shopping lists | P, L and F requirements |
| Q5 | How important is kitshn? | Very important | Section 5, first milestone |
| Q6 | Rewrite in place or new start? | A new branch built as a radical overhaul, replacing the old code when finished. The old version was never published, so no compatibility or data migration is owed. Old code can be reused where useful | Rewrite branch. The tracker and old PRs are cleaned up when it replaces `main` |
| Q7 | Which section 6 changes? | The owner left the architecture to this draft and wants stability | One app container, cookie sessions, integer IDs |
| Q8 | Which data safety items at launch? | Not answered, recommendation used | All of section 3 except DS-6, which can follow soon after |
| Q9 | What does "no suggestions of public users" mean? | Users can't be found: sharing needs a friend or the exact handle | A-15 to A-19, K-8 |
| Q10 | Which kitshn features? | No AI import for now. URL import wasn't ruled out, so it follows the recommendation | R-13 in phase 3 |
| Q11 | Which kitshn build? | Play Store version | K-1 |

| Q12 | Replace PostgreSQL with SQLite? | The owner left it to this draft | Yes, see section 6 |
| Q13 | Which languages? | German is only for the owner personally. Everything must support English and further languages | English is the default and fallback. A new UI language is one file, and catalog names are per-locale rows |

---

## 8. Implementation status

| Area | Status |
|---|---|
| Data safety (section 3) | Done: DS-1 to DS-15, DS-17 to DS-25. DS-16 and DS-22 no longer apply. Version history is in the API, without a UI yet |
| Accounts (A) | Done, except passkeys (A-3) |
| Recipes (R) | Done, except URL import (R-13), recipe books (R-17), Tandoor transfer (R-14) and fractions (R-15). R-12 is left to kitshn |
| Ingredients and units (I) | Done, except I-8 and I-9 |
| Sharing (S) | Done, except the all-recipes link (S-5) |
| Language and UI (U) | Done, except mascot eyes (U-10) |
| kitshn (K) | Read access, login, search, photos and cook log are in, with contract tests. **K-1 is unproven: it hasn't run against the real app yet.** Writing from kitshn isn't done |
| Meal plans, shopping lists, nutrition (P, L, F) | Not started. kitshn gets empty lists |
| MCP (X-2, X-3) | Not started |
