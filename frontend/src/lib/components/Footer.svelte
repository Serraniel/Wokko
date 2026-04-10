<script lang="ts">
  const version = import.meta.env.VITE_APP_VERSION || 'dev';
  const repoUrl = 'https://github.com/Serraniel/Wokko';
</script>

<footer class="border-t border-gray-200 dark:border-gray-700 py-4 mt-auto">
  <div class="max-w-7xl mx-auto px-4">
    <div class="flex flex-wrap items-center justify-center gap-2 text-sm text-gray-500 dark:text-gray-400">
      <a href={repoUrl} target="_blank" rel="noopener noreferrer"
         class="hover:text-wokko-red transition-colors">GitHub</a>
      <span>·</span>
      <a href="{repoUrl}/releases/tag/v{version}" target="_blank" rel="noopener noreferrer"
         class="hover:text-wokko-red transition-colors font-mono">v{version}</a>
      <span>·</span>
      <a href="/changelog" class="hover:text-wokko-red transition-colors">Changelog</a>
      <span>·</span>
      <a href="/license" class="hover:text-wokko-red transition-colors">EUPL-1.2</a>
      <span>·</span>
      <a href="/third-party" class="hover:text-wokko-red transition-colors">Third-Party Notices</a>
    </div>
  </div>
</footer>
