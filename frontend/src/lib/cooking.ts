// Finds times written in a step, such as "30 Minuten köcheln" or "bake for 1 hour", so they
// can become timers. A range counts as its upper end, because that's the safe one to wait for.
const TIME = /(\d+(?:[.,]\d+)?)(?:\s*(?:-|–|bis|to)\s*(\d+(?:[.,]\d+)?))?\s*(stunden|stunde|std\.?|hours?|hrs?|h|minuten|minute|min\.?|mins|minutes)(?![\p{L}])/giu;

export function durationsIn(text: string, stepMinutes = 0): number[] {
	const found = new Set<number>();
	if (stepMinutes > 0) found.add(stepMinutes);
	for (const m of text.matchAll(TIME)) {
		const value = Number((m[2] ?? m[1]).replace(',', '.'));
		const minutes = Math.round(/^(st|h)/i.test(m[3]) ? value * 60 : value);
		if (minutes > 0 && minutes <= 24 * 60) found.add(minutes);
	}
	return [...found];
}
