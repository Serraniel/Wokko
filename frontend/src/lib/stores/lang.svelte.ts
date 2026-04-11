import { browser } from '$app/environment';

function createLangStore() {
  let lang = $state<'en' | 'de'>(
    browser ? ((localStorage.getItem('lang') as 'en' | 'de') || 'en') : 'en'
  );

  function set(l: 'en' | 'de') {
    lang = l;
    if (browser) localStorage.setItem('lang', l);
  }

  return {
    get lang() { return lang; },
    set,
  };
}

export const langStore = createLangStore();
