// To add a language, add src/lib/i18n/messages/<code>.json. Nothing else changes: the file is
// discovered here, listed in the language picker, and checked for completeness by the tests.
// Missing strings fall back to English.
import en from './messages/en.json';

export type MessageKey = Exclude<keyof typeof en, '_language'>;
type Messages = Record<string, string>;

const files = import.meta.glob<{ default: Messages }>('./messages/*.json');
const eager = import.meta.glob<string>('./messages/*.json', { import: '_language', eager: true });

function codeOf(path: string) {
	return path.replace('./messages/', '').replace('.json', '');
}

/** Every language with a messages file, as { code, name }, in the language's own name. */
export const languages = Object.entries(eager)
	.map(([path, name]) => ({ code: codeOf(path), name }))
	.sort((a, b) => a.name.localeCompare(b.name));

let locale = $state('en');
let messages = $state<Messages>(en);

export function currentLocale() {
	return locale;
}

export function bestLocale(wanted: string | undefined | null): string {
	for (const tag of [wanted, ...(typeof navigator === 'undefined' ? [] : navigator.languages)]) {
		const code = tag?.toLowerCase().split('-')[0];
		if (code && languages.some((l) => l.code === code)) return code;
	}
	return 'en';
}

export async function setLocale(code: string) {
	const file = files[`./messages/${code}.json`];
	if (!file) return;
	messages = code === 'en' ? en : (await file()).default;
	locale = code;
	if (typeof document !== 'undefined') document.documentElement.lang = code;
}

/** Translates a key, replacing {name} placeholders. A `count` value selects _one or _other. */
export function t(key: MessageKey | (string & {}), vars?: Record<string, string | number>): string {
	let k = key as string;
	if (vars && typeof vars.count === 'number') {
		const plural = `${k}_${new Intl.PluralRules(locale).select(vars.count)}`;
		k = plural in messages || plural in en ? plural : `${k}_other`;
	}
	let out = messages[k] ?? (en as Messages)[k] ?? k;
	if (vars) for (const [name, value] of Object.entries(vars)) out = out.replaceAll(`{${name}}`, String(value));
	return out;
}

export function formatDate(iso: string, style: 'date' | 'datetime' = 'date'): string {
	if (!iso) return '';
	const d = new Date(iso);
	return new Intl.DateTimeFormat(locale, style === 'date' ? { dateStyle: 'medium' } : { dateStyle: 'medium', timeStyle: 'short' }).format(d);
}

export function formatNumber(n: number): string {
	return new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }).format(n);
}

export function formatBytes(n: number): string {
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let i = 0;
	while (n >= 1000 && i < units.length - 1) {
		n /= 1000;
		i++;
	}
	return `${new Intl.NumberFormat(locale, { maximumFractionDigits: i < 2 ? 0 : 1 }).format(n)} ${units[i]}`;
}

/** "1 h 15 min", in the current language's number format. */
export function formatMinutes(total: number): string {
	if (total < 60) return `${total} min`;
	const h = Math.floor(total / 60);
	const m = total % 60;
	return m ? `${h} h ${m} min` : `${h} h`;
}
