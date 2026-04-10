import { browser } from '$app/environment';

function createThemeStore() {
  let dark = $state(false);

  function init() {
    if (!browser) return;
    const stored = localStorage.getItem('theme');
    dark = stored === 'dark' || (!stored && window.matchMedia('(prefers-color-scheme: dark)').matches);
    apply();
  }

  function toggle() {
    dark = !dark;
    if (browser) {
      localStorage.setItem('theme', dark ? 'dark' : 'light');
      apply();
    }
  }

  function apply() {
    if (!browser) return;
    document.documentElement.classList.toggle('dark', dark);
  }

  return {
    get dark() { return dark; },
    init,
    toggle,
  };
}

export const themeStore = createThemeStore();
