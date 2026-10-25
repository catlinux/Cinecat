// Paletas cálidas de la dirección «Marquesina»: granate, teja, mostaza y
// naranja. El color de cada cartel de respaldo sale del título, así que es
// estable entre cargas y sesiones.
const PALETTES: ReadonlyArray<readonly [string, string]> = [
	['#5a1a1f', '#8f3038'],
	['#3a1013', '#6d2b2f'],
	['#7a3b12', '#c0662a'],
	['#6b4a10', '#c99a2e'],
	['#8c4a17', '#d1682f'],
	['#4a2a12', '#a8681e']
];

export interface PosterColors {
	from: string;
	to: string;
}

// FNV-1a sobre el título normalizado: determinista y sin Math.random.
export function posterColors(title: string): PosterColors {
	let hash = 2166136261;
	for (const character of title.normalize('NFC')) {
		hash ^= character.codePointAt(0) ?? 0;
		hash = Math.imul(hash, 16777619);
	}
	const [from, to] = PALETTES[(hash >>> 0) % PALETTES.length];
	return { from, to };
}
