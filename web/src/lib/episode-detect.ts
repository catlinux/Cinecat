export interface EpisodeGuess {
	season: number | null;
	episode: number | null;
}

const SXXEXX = /s\s*(\d{1,2})\s*[._ -]?\s*e\s*(\d{1,3})(?=\D|$)/i;
const NXNN = /(?:^|\D)(\d{1,2})\s*x\s*(\d{1,3})(?=\D|$)/i;
const TEMPORADA_EPISODIO =
	/\btemporada\s*(\d{1,2})[\s._-]*(?:episodio|cap[ií]tulo|cap\.?|ep\.?)[\s._-]*(\d{1,3})(?=\D|$)/i;
const SOLO_EPISODIO = /\b(?:episodio|cap[ií]tulo|cap\.?|ep\.?)[\s._-]*(\d{1,3})(?=\D|$)/i;
const NUMERO_SUELTO = /(?:^|[\s._-])(\d{1,3})(?=$|[\s._-])/g;

function parseNumber(value: string): number {
	return Number.parseInt(value, 10);
}

// detectEpisode reconoce la temporada y el episodio en el nombre del archivo.
// Si solo aparece el episodio, se usa defaultSeason cuando se informa.
export function detectEpisode(
	fileName: string,
	defaultSeason: number | null = null
): EpisodeGuess {
	let match = SXXEXX.exec(fileName);
	if (match) {
		return { season: parseNumber(match[1]), episode: parseNumber(match[2]) };
	}

	match = NXNN.exec(fileName);
	if (match) {
		return { season: parseNumber(match[1]), episode: parseNumber(match[2]) };
	}

	match = TEMPORADA_EPISODIO.exec(fileName);
	if (match) {
		return { season: parseNumber(match[1]), episode: parseNumber(match[2]) };
	}

	match = SOLO_EPISODIO.exec(fileName);
	if (match) {
		return { season: defaultSeason, episode: parseNumber(match[1]) };
	}

	const stem = fileName.replace(/\.[^./\\]*$/, '');
	let loose: RegExpExecArray | null;
	let last: RegExpExecArray | null = null;
	NUMERO_SUELTO.lastIndex = 0;
	while ((loose = NUMERO_SUELTO.exec(stem)) !== null) {
		last = loose;
	}
	if (last) {
		return { season: defaultSeason, episode: parseNumber(last[1]) };
	}

	return { season: null, episode: null };
}
