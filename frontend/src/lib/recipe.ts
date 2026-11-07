import type { Recipe, RecipeInput } from './api/types.gen.ts';

/** Turns a recipe into the document the save endpoint takes. */
export function toInput(r: Recipe): RecipeInput {
	return {
		name: r.name,
		description: r.description,
		work_minutes: r.work_minutes,
		wait_minutes: r.wait_minutes,
		servings: r.servings,
		servings_text: r.servings_text,
		source_url: r.source_url,
		visibility: r.visibility,
		keywords: r.keywords.map((k) => k.name),
		base_version: r.version,
		steps: r.steps.map((s) => ({
			name: s.name,
			instruction: s.instruction,
			minutes: s.minutes,
			ingredients: s.ingredients.map((i) => ({
				food_id: i.food.id,
				food_name: i.food.name,
				unit_id: i.unit?.id ?? 0,
				unit_name: i.unit?.name ?? '',
				amount: i.amount ?? null,
				note: i.note
			}))
		}))
	};
}
