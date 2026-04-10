<script lang="ts">
  import Footer from '$lib/components/Footer.svelte';
</script>
<div class="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-950">
  <main class="flex-1 max-w-3xl mx-auto w-full px-4 py-12">
    <h1 class="text-3xl font-bold text-gray-900 dark:text-white mb-6">Third-Party Notices</h1>
    <div class="space-y-4 text-sm text-gray-700 dark:text-gray-300">
      <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
        <h2 class="font-semibold text-gray-900 dark:text-white mb-3">Frontend Dependencies</h2>
        <ul class="space-y-2">
          <li><strong>SvelteKit</strong> — MIT License</li>
          <li><strong>TailwindCSS</strong> — MIT License</li>
          <li><strong>Lucide Svelte</strong> — ISC License</li>
          <li><strong>clsx</strong> — MIT License</li>
        </ul>
      </div>
      <p class="text-gray-500 dark:text-gray-400">
        Full license texts are included in the source repository under <code class="font-mono">THIRD-PARTY-NOTICES</code>.
      </p>
    </div>
  </main>
  <Footer />
</div>
