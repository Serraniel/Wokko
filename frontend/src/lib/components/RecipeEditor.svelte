<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, type Recipe, type RecipeDetail, type Ingredient, type Unit } from '$lib/api/client';
  import Button from './Button.svelte';
  import Input from './Input.svelte';
  import { Plus, Trash2, Upload } from 'lucide-svelte';
  import Combobox from './Combobox.svelte';

  interface StepData {
    id?: string;
    description: string;
    duration_min: number | null;
    ingredients: {
      id?: string;
      ingredient_id: string;
      ingredient_name: string;
      unit_id: string;
      unit_name: string;
      amount: string;
      notes: string;
    }[];
  }

  let { recipe = null }: { recipe?: RecipeDetail | null } = $props();

  let name = $state(recipe?.name || '');
  let description = $state(recipe?.description || '');
  let workTime = $state(recipe?.work_time_min?.toString() || '');
  let servings = $state(recipe?.servings?.toString() || '');
  let isPublic = $state(recipe?.is_public || false);
  let tagInput = $state('');
  let tags = $state<string[]>(recipe?.tags || []);
  let steps = $state<StepData[]>(
    recipe?.steps?.map(s => ({
      id: s.id,
      description: s.description,
      duration_min: s.duration_min || null,
      ingredients: s.ingredients?.map(i => ({
        id: i.id,
        ingredient_id: i.ingredient_id,
        ingredient_name: i.ingredient_name || '',
        unit_id: i.unit_id || '',
        unit_name: i.unit_name || '',
        amount: i.amount?.toString() || '',
        notes: i.notes || '',
      })) || [],
    })) || [{ description: '', duration_min: null, ingredients: [] }]
  );

  let photoFile = $state<File | null>(null);
  let photoPreview = $state(recipe?.photo_path ? `/api/media/${recipe.photo_path}` : '');
  let saving = $state(false);
  let error = $state('');

  function addStep() {
    steps = [...steps, { description: '', duration_min: null, ingredients: [] }];
  }

  function removeStep(i: number) {
    steps = steps.filter((_, idx) => idx !== i);
  }

  function addIngredient(stepIdx: number) {
    steps[stepIdx].ingredients = [...steps[stepIdx].ingredients, {
      ingredient_id: '', ingredient_name: '', unit_id: '', unit_name: '', amount: '', notes: ''
    }];
  }

  function removeIngredient(stepIdx: number, ingIdx: number) {
    steps[stepIdx].ingredients = steps[stepIdx].ingredients.filter((_, i) => i !== ingIdx);
  }

  function handlePhotoChange(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    photoFile = file;
    photoPreview = URL.createObjectURL(file);
  }

  function addTag() {
    const t = tagInput.trim();
    if (t && !tags.includes(t)) tags = [...tags, t];
    tagInput = '';
  }

  async function handleSave() {
    if (!name.trim()) { error = 'Recipe name is required'; return; }
    error = '';
    saving = true;
    try {
      const data = {
        name: name.trim(),
        description: description || undefined,
        work_time_min: workTime ? parseInt(workTime) : undefined,
        servings: servings ? parseInt(servings) : undefined,
        is_public: isPublic,
        tags,
      };

      let saved: Recipe;
      if (recipe?.id) {
        saved = await api.updateRecipe(recipe.id, data);
      } else {
        saved = await api.createRecipe(data);
      }

      if (photoFile) {
        await api.uploadRecipePhoto(saved.id, photoFile).catch(console.error);
      }

      for (const [idx, step] of steps.entries()) {
        const savedStep = await api.createStep(saved.id, {
          description: step.description,
          duration_min: step.duration_min || undefined,
          step_order: idx + 1,
        });
        for (const ing of step.ingredients) {
          if (!ing.ingredient_id) continue;
          await api.addStepIngredient(saved.id, savedStep.id, {
            ingredient_id: ing.ingredient_id,
            unit_id: ing.unit_id || undefined,
            amount: ing.amount ? parseFloat(ing.amount) : undefined,
            notes: ing.notes || undefined,
          }).catch(console.error);
        }
      }

      goto(`/recipes/${saved.id}`);
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }
</script>

<div class="max-w-3xl mx-auto space-y-8">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
      {recipe ? 'Edit Recipe' : 'New Recipe'}
    </h1>
    <Button onclick={handleSave} loading={saving}>{recipe ? 'Save Changes' : 'Create Recipe'}</Button>
  </div>

  {#if error}<p class="text-sm text-red-600 bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">{error}</p>{/if}

  <!-- Basic info -->
  <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 space-y-4">
    <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Info</h2>
    <Input label="Recipe Name" bind:value={name} placeholder="e.g. Spaghetti Bolognese" required />
    <div>
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
      <textarea bind:value={description} placeholder="A brief description of the recipe..."
        rows="3"
        class="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800
               text-gray-900 dark:text-gray-100 px-3 py-2 text-sm
               focus:outline-none focus:ring-2 focus:ring-wokko-red resize-none"
      ></textarea>
    </div>
    <div class="grid grid-cols-2 gap-4">
      <Input label="Work Time (minutes)" type="number" bind:value={workTime} placeholder="30" />
      <Input label="Servings" type="number" bind:value={servings} placeholder="4" />
    </div>
    <div class="flex items-center gap-3">
      <input type="checkbox" id="is-public" bind:checked={isPublic} class="rounded border-gray-300 text-wokko-red focus:ring-wokko-red" />
      <label for="is-public" class="text-sm text-gray-700 dark:text-gray-300">Make this recipe public</label>
    </div>
  </div>

  <!-- Photo -->
  <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
    <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Photo</h2>
    {#if photoPreview}
      <img src={photoPreview} alt="Preview" class="w-full h-48 object-cover rounded-lg mb-3" />
    {/if}
    <label class="cursor-pointer flex items-center gap-2 px-4 py-2 border-2 border-dashed border-gray-300 dark:border-gray-600
                  rounded-lg text-sm text-gray-500 hover:border-wokko-red transition-colors w-full justify-center">
      <Upload class="w-4 h-4" />
      {photoPreview ? 'Change photo' : 'Upload photo'}
      <input type="file" accept="image/*" onchange={handlePhotoChange} class="hidden" />
    </label>
  </div>

  <!-- Tags -->
  <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
    <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Tags</h2>
    <div class="flex gap-2 mb-3 flex-wrap">
      {#each tags as tag}
        <span class="flex items-center gap-1 px-3 py-1 rounded-full bg-gray-100 dark:bg-gray-700 text-sm">
          {tag}
          <button type="button" onclick={() => tags = tags.filter(t => t !== tag)} class="text-gray-400 hover:text-red-500">×</button>
        </span>
      {/each}
    </div>
    <div class="flex gap-2">
      <input bind:value={tagInput} placeholder="Add tag..."
        onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addTag(); } }}
        class="flex-1 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800
               text-gray-900 dark:text-gray-100 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-wokko-red" />
      <Button type="button" variant="secondary" onclick={addTag}>Add</Button>
    </div>
  </div>

  <!-- Steps -->
  <div class="space-y-4">
    <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Steps</h2>
    {#each steps as step, stepIdx}
      <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 space-y-4">
        <div class="flex items-center justify-between">
          <h3 class="font-semibold text-gray-900 dark:text-white">Step {stepIdx + 1}</h3>
          <Button type="button" variant="ghost" onclick={() => removeStep(stepIdx)}>
            <Trash2 class="w-4 h-4 text-red-500" />
          </Button>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Instructions</label>
          <textarea bind:value={step.description} placeholder="Describe this step..." rows="3"
            class="w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800
                   text-gray-900 dark:text-gray-100 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-wokko-red resize-none"
          ></textarea>
        </div>
        <Input label="Duration (minutes)" type="number" value={step.duration_min?.toString() || ''}
          onchange={(v) => { step.duration_min = v ? parseInt(v) : null; }}
          placeholder="Optional" class="max-w-xs" />

        <!-- Step ingredients -->
        <div class="space-y-3">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300">Ingredients for this step</h4>
          {#each step.ingredients as ing, ingIdx}
            <div class="grid grid-cols-[2fr_1fr_80px_auto] gap-2 items-end">
              <Combobox
                placeholder="Ingredient..."
                value={ing.ingredient_id ? { id: ing.ingredient_id, label: ing.ingredient_name } : null}
                onSearch={async (q) => {
                  const list = await api.searchIngredients(q, 8).catch(() => []);
                  return list.map(i => ({ id: i.id, label: i.name_en }));
                }}
                onSelect={(opt) => {
                  ing.ingredient_id = opt?.id ?? '';
                  ing.ingredient_name = opt?.label ?? '';
                }}
                onCreate={async (name) => {
                  const created = await api.createIngredient({ name_en: name });
                  return { id: created.id, label: created.name_en };
                }}
              />
              <Combobox
                placeholder="Unit..."
                value={ing.unit_id ? { id: ing.unit_id, label: ing.unit_name } : null}
                onSearch={async (q) => {
                  const list = await api.searchUnits(q, 8).catch(() => []);
                  return list.map(u => ({ id: u.id, label: u.abbreviation_en || u.name_en }));
                }}
                onSelect={(opt) => {
                  ing.unit_id = opt?.id ?? '';
                  ing.unit_name = opt?.label ?? '';
                }}
                onCreate={async (name) => {
                  const created = await api.createUnit({ name_en: name, abbreviation_en: name });
                  return { id: created.id, label: created.abbreviation_en || created.name_en };
                }}
              />
              <input bind:value={ing.amount} type="number" step="0.01" placeholder="Qty"
                class="rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800
                       text-gray-900 dark:text-gray-100 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-wokko-red" />
              <button type="button" onclick={() => removeIngredient(stepIdx, ingIdx)}
                class="p-2 text-red-400 hover:text-red-600 transition-colors">
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
          {/each}
          <Button type="button" variant="ghost" onclick={() => addIngredient(stepIdx)}>
            <Plus class="w-4 h-4" /> Add ingredient
          </Button>
        </div>
      </div>
    {/each}
    <Button type="button" variant="secondary" onclick={addStep} class="w-full">
      <Plus class="w-4 h-4" /> Add Step
    </Button>
  </div>

  <div class="flex justify-end gap-3 pb-8">
    <a href={recipe ? `/recipes/${recipe.id}` : '/recipes'}>
      <Button variant="secondary">Cancel</Button>
    </a>
    <Button onclick={handleSave} loading={saving}>{recipe ? 'Save Changes' : 'Create Recipe'}</Button>
  </div>
</div>
