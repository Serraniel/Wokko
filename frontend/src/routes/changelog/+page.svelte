<script lang="ts">
  import Footer from '$lib/components/Footer.svelte';
  const version = import.meta.env.VITE_APP_VERSION || 'dev';
</script>
<div class="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-950">
  <main class="flex-1 max-w-3xl mx-auto w-full px-4 py-12">
    <h1 class="text-3xl font-bold text-gray-900 dark:text-white mb-6">Changelog</h1>
    <div class="space-y-4">
      <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">v{version}</h2>
        <p class="text-gray-500 dark:text-gray-400 text-sm mt-1">Current version</p>
        <ul class="mt-3 space-y-1 text-sm text-gray-700 dark:text-gray-300 list-disc list-inside">
          <li>Initial release of Wokko</li>
          <li>Recipe management with steps and ingredients</li>
          <li>Admin panel with user and invite management</li>
          <li>Public recipe gallery</li>
          <li>Dark/light mode support</li>
        </ul>
      </div>
    </div>
  </main>
  <Footer />
</div>
