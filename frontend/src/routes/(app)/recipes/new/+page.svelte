<script lang="ts">
	import RecipeEditor from '#lib/components/RecipeEditor.svelte';
	import { t } from '#lib/i18n/index.svelte.ts';
</script>

<svelte:head><title>{t('editor.new_title')} · Wokko</title></svelte:head>

<RecipeEditor />
