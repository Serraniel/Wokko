// Display preferences for this device. They change how things look, never what's stored.
export type AmountStyle = 'decimal' | 'fraction';

function read(): AmountStyle {
	try {
		return localStorage.getItem('wokko-amounts') === 'fraction' ? 'fraction' : 'decimal';
	} catch {
		return 'decimal';
	}
}

class Prefs {
	amounts = $state<AmountStyle>(typeof localStorage === 'undefined' ? 'decimal' : read());

	setAmounts(style: AmountStyle) {
		this.amounts = style;
		try {
			localStorage.setItem('wokko-amounts', style);
		} catch {
			// The choice still applies until the page reloads.
		}
	}
}

export const prefs = new Prefs();
