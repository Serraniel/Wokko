# Language Preference (EN/DE) — Design

**Goal:** Let users switch between English and German for ingredient and unit names throughout the app (search autocomplete, recipe detail view). UI chrome stays English for now (full i18n is issue #22).

**Architecture:** A `langStore` Svelte rune store persists `'en' | 'de'` in localStorage. The `apiRequest` function sends `Accept-Language: de` when DE is active. The backend reads the header and selects the appropriate name fields from the DB (using `COALESCE(name_de, name_en)` for DE). A toggle button (EN / DE) is added to the Nav next to the theme toggle.

**Scope:** Data-language only. Affected surfaces: ingredient/unit combobox labels in the recipe editor, ingredient names in recipe detail view. Not in scope: UI string translation.

---

## Surfaces affected

| Surface | Current | After |
|---|---|---|
| Ingredient combobox (editor) | `name_en` | `name_de \|\| name_en` when DE |
| Unit combobox (editor) | `abbreviation_en \|\| name_en` | `abbreviation_de \|\| name_de \|\| abbreviation_en \|\| name_en` when DE |
| Recipe detail ingredient list | `ingredient_name` (always EN from JOIN) | resolved by backend from `Accept-Language` |

## Backend changes

**`backend/internal/db/queries/recipes.sql`** — extend `ListStepIngredients` to also select DE variants:
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

**`backend/internal/db/queries/models.go`** — add 3 new fields to `StepIngredient`:
```go
IngredientNameDe string `json:"ingredient_name_de"`
UnitNameDe       string `json:"unit_name_de"`
UnitAbbrDe       string `json:"unit_abbr_de"`
```

**`backend/internal/db/queries/recipes.sql.go`** — update the embedded SQL string and the `rows.Scan(...)` call to include the 3 new fields.

**`backend/internal/api/recipe_handler.go`** — `buildIngredientResponses` gets a `lang string` param; picks DE fields when `lang == "de"`. Both call sites pass `c.Request().Header.Get("Accept-Language")` (normalized to `"de"` or `"en"`).

## Frontend changes

**`frontend/src/lib/stores/lang.svelte.ts`** — new file:
```ts
import { browser } from '$app/environment';

function createLangStore() {
  let lang = $state<'en' | 'de'>(
    browser ? (localStorage.getItem('lang') as 'en' | 'de' || 'en') : 'en'
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

**`frontend/src/lib/api/client.ts`** — import `langStore`, add `Accept-Language` header in `doFetch`:
```ts
if (typeof localStorage !== 'undefined') {
  headers['Accept-Language'] = langStore.lang;
}
```

**`frontend/src/lib/components/Nav.svelte`** — add toggle next to `<ThemeToggle />` in both desktop and mobile menus.

**`frontend/src/lib/components/RecipeEditor.svelte`** — Combobox `onSearch` callbacks use `langStore.lang` to pick the right label from the API response.
