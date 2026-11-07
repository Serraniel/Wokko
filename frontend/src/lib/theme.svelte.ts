export type ThemeChoice = 'system' | 'light' | 'dark';

function read(): ThemeChoice {
	try {
		const v = localStorage.getItem('wokko-theme');
		if (v === 'light' || v === 'dark') return v;
	} catch {
		// Storage can be blocked. The system theme still works.
	}
	return 'system';
}

class Theme {
	choice = $state<ThemeChoice>(typeof localStorage === 'undefined' ? 'system' : read());

	set(next: ThemeChoice) {
		this.choice = next;
		try {
			if (next === 'system') localStorage.removeItem('wokko-theme');
			else localStorage.setItem('wokko-theme', next);
		} catch {
			// The choice still applies until the page reloads.
		}
		if (next === 'system') delete document.documentElement.dataset.theme;
		else document.documentElement.dataset.theme = next;
	}
}

export const theme = new Theme();
