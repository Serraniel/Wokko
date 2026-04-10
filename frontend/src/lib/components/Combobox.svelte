<script lang="ts">
  interface Option { id: string; label: string; }
  interface Props {
    label?: string;
    placeholder?: string;
    value?: Option | null;
    onSearch: (q: string) => Promise<Option[]>;
    onSelect: (opt: Option | null) => void;
    onCreate?: (name: string) => Promise<Option>;
  }
  let { label, placeholder = 'Search...', value = null, onSearch, onSelect, onCreate }: Props = $props();

  let query = $state(value?.label || '');
  let results = $state<Option[]>([]);
  let open = $state(false);
  let loading = $state(false);
  let timer: ReturnType<typeof setTimeout>;

  async function handleInput(q: string) {
    query = q;
    if (!q.trim()) { results = []; open = false; onSelect(null); return; }
    loading = true;
    clearTimeout(timer);
    timer = setTimeout(async () => {
      results = await onSearch(q);
      open = true;
      loading = false;
    }, 200);
  }

  async function pick(opt: Option) {
    query = opt.label;
    open = false;
    onSelect(opt);
  }

  async function createNew() {
    if (!onCreate) return;
    const opt = await onCreate(query);
    await pick(opt);
  }
</script>

<div class="relative flex flex-col gap-1">
  {#if label}
    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</label>
  {/if}
  <input
    type="text"
    bind:value={query}
    {placeholder}
    oninput={(e) => handleInput((e.target as HTMLInputElement).value)}
    onfocus={() => { if (results.length) open = true; }}
    onblur={() => setTimeout(() => open = false, 150)}
    class="w-full rounded-lg border px-3 py-2 text-sm border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-wokko-red"
  />
  {#if open && (results.length > 0 || (onCreate && query.trim()))}
    <div class="absolute top-full left-0 right-0 z-50 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg mt-1 max-h-48 overflow-y-auto">
      {#each results as opt}
        <button type="button" onclick={() => pick(opt)}
          class="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700">
          {opt.label}
        </button>
      {/each}
      {#if onCreate && query.trim() && !results.find(r => r.label.toLowerCase() === query.toLowerCase())}
        <button type="button" onclick={createNew}
          class="w-full text-left px-3 py-2 text-sm text-wokko-red hover:bg-gray-100 dark:hover:bg-gray-700">
          + Create "{query}"
        </button>
      {/if}
    </div>
  {/if}
</div>
