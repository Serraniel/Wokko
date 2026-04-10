<script lang="ts">
  let { size = 40, class: className = '' } = $props();
</script>

<img src="/logo.svg" alt="Wokko" width={size} height={size} class={className} />
