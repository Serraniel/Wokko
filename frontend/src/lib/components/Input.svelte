<script lang="ts">
  interface Props {
    label?: string;
    type?: string;
    placeholder?: string;
    value?: string;
    error?: string;
    required?: boolean;
    disabled?: boolean;
    autofocus?: boolean;
    class?: string;
    id?: string;
    autocomplete?: string;
    onchange?: (value: string) => void;
  }
  let { label, type = 'text', placeholder = '', value = $bindable(''),
        error, required = false, disabled = false, autofocus = false, class: className = '', id, autocomplete, onchange }: Props = $props();

  const inputId = id || `input-${Math.random().toString(36).slice(2)}`;
</script>

<div class="flex flex-col gap-1 {className}">
  {#if label}
    <label for={inputId} class="text-sm font-medium text-gray-700 dark:text-gray-300">
      {label}{#if required}<span class="text-wokko-red ml-1">*</span>{/if}
    </label>
  {/if}
  <input
    {type}
    id={inputId}
    bind:value
    {placeholder}
    {required}
    {disabled}
    {autofocus}
    autocomplete={autocomplete as any}
    oninput={(e) => onchange?.((e.target as HTMLInputElement).value)}
    class="w-full rounded-lg border px-3 py-2 text-sm transition-colors
           border-gray-300 dark:border-gray-600
           bg-white dark:bg-gray-800
           text-gray-900 dark:text-gray-100
           placeholder-gray-400 dark:placeholder-gray-500
           focus:outline-none focus:ring-2 focus:ring-wokko-red focus:border-transparent
           disabled:opacity-50 disabled:cursor-not-allowed
           {error ? 'border-red-500 focus:ring-red-500' : ''}"
  />
  {#if error}
    <p class="text-sm text-red-600 dark:text-red-400">{error}</p>
  {/if}
</div>
