# Kitchen knowledge: built-in recipe insights without an AI model

Status: plan, not started. Tracking issue: #105.

## Goal

Wokko should understand recipes the way an experienced cook reads them. For example:

- "This sauce is a velouté, finished with an egg yolk liaison (Allemande-style)."
- "Hot roux plus cold stock avoids lumps."
- "Contains celery, egg, milk and gluten."

It should also point out likely problems: missing amounts, ingredients that no step uses, risky steps.

All of this runs inside Wokko: no cloud or local LLM, and no network. It has to work the same in every language, and it must never change a recipe by itself.

## Where it works

The analysis reads the **saved, structured recipe**: foods from the catalog, units, amounts and step text. It does not read transcripts or web pages. An assistant or the importer turns those into a recipe first, so each source arrives in the same shape.

This keeps the knowledge independent of where a recipe came from. It also lets the analysis rely on catalog IDs instead of guessing at words.

It runs:
- on the recipe page, as an "Insights" section
- in the editor, as hints while editing
- through a read-only MCP tool, `get_recipe_insights`, so an assistant can explain them or fix problems with the person's consent

Results are computed on the fly. This is cheap: rules over one recipe take well under a millisecond. Nothing is stored, so improving the knowledge improves old recipes too.

## Knowledge pack

Knowledge is **data shipped in the binary**, separate from the database. It lives as files in `backend/internal/knowledge/data/`, embedded with `go:embed`. It is versioned and reviewed like code, and updated with every release.

Why it doesn't go into the catalog:
- The catalog belongs to the people using Wokko and changes through migrations only once.
- The knowledge pack needs to improve with every release without touching user data.

It has three layers:

1. **Food classes.** These are concepts such as `fat.butter`, `starch.wheat_flour`, `liquid.stock.poultry`, `liquid.milk`, `dairy.cream`, `egg.yolk`, `acid.caper`, `allergen.celery`.
   - Seed foods are mapped by their stable ID.
   - Foods people created are mapped through a per-locale name lexicon, then through their aliases.
   - Admins can set a food's classes by hand. That's a small catalog addition: one table, a migration, and an editor field.
2. **Technique lexicon.** These are per-locale words and patterns that mark a technique in step text, matched on the normalized text.
   - Examples include `Mehlschwitze`, `roux`, `legieren`, `liaison`, `ablöschen`, `deglaze`, `blanchieren`, `kalt ansetzen`, `abschäumen`, `reduzieren`, `rösten`, `aufkochen`.
   - English and German ship first. A missing language falls back to the ingredient-only rules.
3. **Rules.** Each rule is a small declarative entry: conditions over classes, amounts, units, step order and techniques. It produces a finding with a severity, a localized title and explanation, and the steps it refers to.

Rules are written in Go, with tables in the data files. A rule language is not worth it at this size.

## What it finds, first version

**Sauce families.** The five mother sauces and their best-known derivatives:

| Found | Means |
|---|---|
| fat + wheat flour in a step, then stock | Velouté |
| fat + flour, then milk | Béchamel. With cheese, Mornay |
| roux (darker, or browned bones and tomato paste), then brown stock | Espagnole |
| egg yolk + butter, warm, with an acid | Hollandaise. With tarragon and shallot, Béarnaise. With tomato paste as well, Choron |
| tomatoes cooked down | Tomato sauce |
| velouté + cream | Suprême |
| velouté + yolk and cream liaison | Allemande |

The roux ratio is checked from amounts converted to grams (equal parts by weight is the classic). Color comes from browning cues: roasting, tomato paste, or "dunkel"/"brown".

**Techniques with tips.** Each one gets a short explanation card, for example:
- roux: hot roux plus cold liquid, or the other way round, and boil it once because starch binds from about 60 °C
- liaison: yolks need starch or gentle heat
- stock: start cold, skim, season at the end when it reduces
- Maillard browning above about 140 °C
- blanching and shocking keeps green vegetables green

**Allergens and diets.** These are the EU's 14 allergens, worked out from food classes: gluten, crustaceans, egg, fish, peanut, soy, milk, nuts, celery, mustard, sesame, sulphites, lupin and molluscs. Vegetarian and vegan follow from the same classes.

This has the most everyday value. It's also where wrong answers matter most, so it reads "contains, as far as Wokko knows", and lists foods it couldn't classify.

**Problem hints (a recipe linter).** Each hint is info or a warning, never a block:
- an ingredient that no step uses, or one a step names that isn't listed
- an amount without a unit where one is usual (`200 Mehl`), or a unit that doesn't fit the food (`ml` of flour)
- step minutes adding up to far more than the recipe's time
- salt added before a long reduction ("season at the end")
- a hard boil after an egg yolk liaison without starch. With flour in the sauce it's a note, not a warning
- an acid added to milk or cream over heat without a starch (curdling)
- poultry or minced meat without a step that cooks it
- yeast and salt mixed directly. This one is low priority

### Hühnerfrikassee from the example transcript

Saved as a recipe, the analysis would show:
- **Chicken stock**: light, cold start, skimmed. The roasted onion and ginger give it a golden color.
- **Sauce**: velouté (butter and flour roux plus stock), refined with cream and yolk into an Allemande-style sauce.
- **Tips** matching the video: cold stock into hot roux, and season after reducing.
- **Allergens**: celery, egg, milk, gluten.
- **No warnings**: the boil after the liaison is fine because the flour protects the yolks.

## Machine learning, honestly

Rules come first. They are explainable, testable and work from day one.

Learned models are worth it later for two things:
- matching free-text food names to food classes
- spotting techniques in step text written in unusual ways

The plan for both is a small model trained when Wokko is built, not at runtime, and shipped as a few hundred KB of weights. A character n-gram naive Bayes or logistic regression would do: pure Go, no GPU, deterministic.

What's missing is labeled training data under a license Wokko can ship. Open sources such as Wikidata (CC0) and Open Food Facts (ODbL, share-alike, which needs care) can provide food classes and allergens. The models get built when the rules plainly miss things in real recipes, not before.

## Order of work

1. Food classes for the seed catalog, the lexicon for user foods, and the admin field, plus **allergens and diets**. This is useful right away and the foundation for everything else.
2. The recipe linter's structural checks: unused ingredients, units, times.
3. Technique detection and tip cards, in English and German.
4. Sauce families.
5. The MCP tool `get_recipe_insights`, and editor hints.
6. Later, and only if needed: learned matching.

## Also worth considering

- **Substitutions**: buttermilk is milk plus lemon juice, and self-raising flour is flour plus baking powder. These come from the same food classes.
- **Seasonality**: when produce is in season locally, by region. It's a small table, per hemisphere.
- **Related recipes**: "other recipes with a velouté" comes free once sauces are found.
- **Scaling warnings**: eggs that don't divide, and times that don't scale with the amount.

## Not in scope

Generating recipes or text, calling an AI model, changing recipes automatically, and health claims beyond allergens and diets.
