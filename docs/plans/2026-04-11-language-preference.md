# Language Preference (EN/DE) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add EN/DE language toggle that switches ingredient and unit names throughout the app (search autocomplete + recipe detail).

**Architecture:** `langStore` Svelte rune in localStorage → `Accept-Language` header on every API request → backend reads header and returns language-resolved ingredient/unit names. Frontend Combobox also picks the right field client-side for search results.

**Tech Stack:** SvelteKit (Svelte 5 runes), Go/Echo, PostgreSQL, sqlc-generated code (manually updated — sqlc binary not available).

---

### Task 1: Extend backend DB model with DE name fields

**Files:**
- Modify: `backend/internal/db/queries/recipes.sql`
- Modify: `backend/internal/db/queries/models.go`
- Modify: `backend/internal/db/queries/recipes.sql.go`

**Step 1: Update the SQL query**

In `backend/internal/db/queries/recipes.sql`, replace the `ListStepIngredients` query:

```sql
-- name: ListStepIngredients :many
SELECT
    si.id, si.step_id, si.ingredient_id, si.unit_id, si.amount, si.notes,
    i.name_en AS ingredient_name,
    COALESCE(i.name_de, '') AS ingredient_name_de,
    COALESCE(u.name_en, '') AS unit_name,
    COALESCE(u.name_de, '') AS unit_name_de,
    COALESCE(u.abbreviation_en, '') AS unit_abbr,
    COALESCE(u.abbreviation_de, '') AS unit_abbr_de
FROM step_ingredients si
JOIN ingredients i ON i.id = si.ingredient_id
LEFT JOIN units u ON u.id = si.unit_id
WHERE si.step_id = $1;
```

**Step 2: Add DE fields to the StepIngredient struct**

In `backend/internal/db/queries/models.go`, find the `StepIngredient` struct (around line 108) and add 3 fields after `UnitAbbr`:

```go
IngredientNameDe string         `json:"ingredient_name_de"`
UnitNameDe       string         `json:"unit_name_de"`
UnitAbbrDe       string         `json:"unit_abbr_de"`
```

**Step 3: Update the generated Go scan**

In `backend/internal/db/queries/recipes.sql.go`, find the `listStepIngredients` const (around line 466) and replace the entire block with:

```go
const listStepIngredients = `-- name: ListStepIngredients :many
SELECT
    si.id, si.step_id, si.ingredient_id, si.unit_id, si.amount, si.notes,
    i.name_en AS ingredient_name,
    COALESCE(i.name_de, '') AS ingredient_name_de,
    COALESCE(u.name_en, '') AS unit_name,
    COALESCE(u.name_de, '') AS unit_name_de,
    COALESCE(u.abbreviation_en, '') AS unit_abbr,
    COALESCE(u.abbreviation_de, '') AS unit_abbr_de
FROM step_ingredients si
JOIN ingredients i ON i.id = si.ingredient_id
LEFT JOIN units u ON u.id = si.unit_id
WHERE si.step_id = $1
`
```

And update the `rows.Scan(...)` call inside `ListStepIngredients` to add the 3 new fields at the end:

```go
if err := rows.Scan(
    &i.ID,
    &i.StepID,
    &i.IngredientID,
    &i.UnitID,
    &i.Amount,
    &i.Notes,
    &i.IngredientName,
    &i.IngredientNameDe,
    &i.UnitName,
    &i.UnitNameDe,
    &i.UnitAbbr,
    &i.UnitAbbrDe,
); err != nil {
    return nil, err
}
```

**Step 4: Build to verify**

```bash
cd backend && go build ./...
```

Expected: no errors.

**Step 5: Run backend tests**

```bash
cd backend && go test ./...
```

Expected: all pass.

**Step 6: Commit**

```bash
git -c commit.gpgsign=false commit -am "feat(i18n): extend ListStepIngredients to return DE name variants"
```

---

### Task 2: Update backend handler to resolve language from Accept-Language header

**Files:**
- Modify: `backend/internal/api/recipe_handler.go`
- Modify: `backend/internal/api/public_handler.go`

**Step 1: Add a lang helper function**

At the top of `recipe_handler.go`, after the imports, add this helper:

```go
// resolveLang returns "de" if the Accept-Language header starts with "de", else "en".
func resolveLang(c echo.Context) string {
    al := c.Request().Header.Get("Accept-Language")
    if len(al) >= 2 && al[:2] == "de" {
        return "de"
    }
    return "en"
}
```

**Step 2: Update buildIngredientResponses signature**

Change the function signature to accept a `lang` parameter:

```go
func buildIngredientResponses(lang string, ings []dbq.StepIngredient) []StepIngredientResponse {
```

Inside the loop, replace the three hardcoded field assignments:

```go
IngredientName: ing.IngredientName,
UnitName:       ing.UnitName,
UnitAbbr:       ing.UnitAbbr,
```

With language-aware selection:

```go
IngredientName: pickLang(lang, ing.IngredientName, ing.IngredientNameDe),
UnitName:       pickLang(lang, ing.UnitName, ing.UnitNameDe),
UnitAbbr:       pickLang(lang, ing.UnitAbbr, ing.UnitAbbrDe),
```

Also add the `pickLang` helper below `resolveLang`:

```go
// pickLang returns deVal if lang=="de" and deVal is non-empty, else enVal.
func pickLang(lang, enVal, deVal string) string {
    if lang == "de" && deVal != "" {
        return deVal
    }
    return enVal
}
```

**Step 3: Update call sites in recipe_handler.go**

Find both calls to `buildIngredientResponses(ings)` (around lines 236 and 426) and update them to pass `resolveLang(c)`:

```go
ingResps := buildIngredientResponses(resolveLang(c), ings)
```

**Step 4: Update call site in public_handler.go**

Find the call to `buildIngredientResponses(ings)` in `public_handler.go` and update it:

```go
ingResps := buildIngredientResponses(resolveLang(c), ings)
```

**Step 5: Build and test**

```bash
cd backend && go build ./... && go test ./...
```

Expected: all pass.

**Step 6: Commit**

```bash
git -c commit.gpgsign=false commit -am "feat(i18n): resolve ingredient/unit names from Accept-Language header in recipe responses"
```

---

### Task 3: Create frontend lang store

**Files:**
- Create: `frontend/src/lib/stores/lang.svelte.ts`

**Step 1: Create the store**

```ts
import { browser } from '$app/environment';

function createLangStore() {
  let lang = $state<'en' | 'de'>(
    browser ? ((localStorage.getItem('lang') as 'en' | 'de') || 'en') : 'en'
  );

  function set(l: 'en' | 'de') {
    lang = l;
    if (browser) localStorage.setItem('lang', l);
  }

  return {
    get lang() { return lang; },
    set,
  };
}

export const langStore = createLangStore();
```

**Step 2: Verify TypeScript compiles**

```bash
cd frontend && npm run build 2>&1 | grep -E "error|Error|built in"
```

Expected: `✓ built in ...`

**Step 3: Commit**

```bash
git -c commit.gpgsign=false commit -am "feat(i18n): add lang store persisted to localStorage"
```

---

### Task 4: Send Accept-Language header in all API requests

**Files:**
- Modify: `frontend/src/lib/api/client.ts`

**Step 1: Import langStore**

At the top of `client.ts`, add:

```ts
import { browser } from '$app/environment';
import { langStore } from '$lib/stores/lang.svelte';
```

**Step 2: Add header in doFetch**

In the `doFetch` function, after the Authorization header block, add:

```ts
if (browser) {
  headers['Accept-Language'] = langStore.lang;
}
```

**Step 3: Build to verify**

```bash
cd frontend && npm run build 2>&1 | grep -E "error|Error|built in"
```

Expected: `✓ built in ...`

**Step 4: Commit**

```bash
git -c commit.gpgsign=false commit -am "feat(i18n): send Accept-Language header on all API requests based on lang store"
```

---

### Task 5: Add language toggle to Nav

**Files:**
- Modify: `frontend/src/lib/components/Nav.svelte`

**Step 1: Import langStore**

Add to the `<script>` block:

```ts
import { langStore } from '$lib/stores/lang.svelte';
```

**Step 2: Add toggle in desktop nav**

After `<ThemeToggle />` in the desktop nav section:

```svelte
<button
  onclick={() => langStore.set(langStore.lang === 'en' ? 'de' : 'en')}
  class="text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-wokko-red transition-colors w-8 text-center"
  title="Switch language"
>
  {langStore.lang === 'en' ? 'DE' : 'EN'}
</button>
```

The button shows the language you would switch TO (so when EN is active, show "DE" = "switch to DE").

**Step 3: Add toggle in mobile nav**

In the mobile menu section, after the Theme row, add:

```svelte
<div class="flex items-center justify-between py-2">
  <span class="text-gray-600 dark:text-gray-300 text-sm">Language</span>
  <button
    onclick={() => langStore.set(langStore.lang === 'en' ? 'de' : 'en')}
    class="text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-wokko-red transition-colors"
  >
    {langStore.lang === 'en' ? 'Deutsch' : 'English'}
  </button>
</div>
```

**Step 4: Build to verify**

```bash
cd frontend && npm run build 2>&1 | grep -E "error|Error|built in"
```

**Step 5: Commit**

```bash
git -c commit.gpgsign=false commit -am "feat(i18n): add EN/DE language toggle to Nav"
```

---

### Task 6: Use lang-aware labels in RecipeEditor Combobox

**Files:**
- Modify: `frontend/src/lib/components/RecipeEditor.svelte`

**Step 1: Import langStore**

Add to the `<script>` block:

```ts
import { langStore } from '$lib/stores/lang.svelte';
```

**Step 2: Update ingredient Combobox onSearch**

Replace the current `onSearch` for ingredients:

```ts
onSearch={async (q) => {
  const list = await api.searchIngredients(q, 8).catch(() => []);
  return list.map(i => ({
    id: i.id,
    label: langStore.lang === 'de'
      ? (i.name_de || i.name_en)
      : i.name_en,
  }));
}}
```

**Step 3: Update unit Combobox onSearch**

Replace the current `onSearch` for units:

```ts
onSearch={async (q) => {
  const list = await api.searchUnits(q, 8).catch(() => []);
  return list.map(u => ({
    id: u.id,
    label: langStore.lang === 'de'
      ? (u.abbreviation_de || u.name_de || u.abbreviation_en || u.name_en)
      : (u.abbreviation_en || u.name_en),
  }));
}}
```

**Step 4: Update Unit type in client.ts to include DE fields**

In `frontend/src/lib/api/client.ts`, find the `Unit` interface and add DE fields:

```ts
export interface Unit {
  id: string;
  name_en: string;
  name_de?: string;
  abbreviation_en?: string;
  abbreviation_de?: string;
}
```

Also update the `Ingredient` interface to surface `name_de`:

```ts
export interface Ingredient {
  id: string;
  name_en: string;
  name_de?: string;
  plural_en?: string;
  plural_de?: string;
  default_unit_id?: string;
  suggested_unit_id?: string;
}
```

(These fields are already returned by the backend — just not typed on the frontend.)

**Step 5: Build to verify**

```bash
cd frontend && npm run build 2>&1 | grep -E "error|Error|built in"
```

Expected: `✓ built in ...`

**Step 6: Commit**

```bash
git -c commit.gpgsign=false commit -am "feat(i18n): use lang-aware labels in ingredient and unit Combobox"
```

---

### Task 7: Build host, rebuild Docker, open PR

**Step 1: Final host build**

```bash
cd frontend && npm run build
```

**Step 2: Rebuild Docker and restart**

```bash
docker compose -f docker-compose.local.yml build frontend && docker compose -f docker-compose.local.yml up -d --force-recreate frontend
```

**Step 3: Smoke test**
- Open http://localhost:3002, switch to DE in Nav
- Open a recipe — ingredient names should appear in German if a DE translation exists
- Open recipe editor, type ingredient — Combobox should show DE names

**Step 4: Push and create PR**

```bash
git push origin feat/language-preference
gh pr create --repo Serraniel/Wokko \
  --title "feat(i18n): EN/DE language preference for ingredient and unit names" \
  --body "Adds a language toggle (EN/DE) to the Nav. Preference is stored in localStorage and sent as Accept-Language on every API request. Backend resolves ingredient and unit names in the requested language for recipe detail views. Combobox in the recipe editor also shows names in the selected language. Closes #23."
```
