export interface RowScrollState {
	canPrev: boolean;
	canNext: boolean;
	prevOffset: number;
	nextOffset: number;
	overflow: boolean;
}

const EDGE_TOLERANCE = 1;

export function rowScrollState(
	scrollLeft: number,
	clientWidth: number,
	scrollWidth: number
): RowScrollState {
	const left = Number.isFinite(scrollLeft) ? Math.max(0, scrollLeft) : 0;
	const width = Number.isFinite(clientWidth) ? Math.max(0, clientWidth) : 0;
	const total = Number.isFinite(scrollWidth) ? Math.max(0, scrollWidth) : 0;
	const maxScroll = Math.max(0, total - width);
	const overflow = maxScroll > EDGE_TOLERANCE;
	const page = width * 0.9;
	return {
		canPrev: overflow && left > EDGE_TOLERANCE,
		canNext: overflow && left < maxScroll - EDGE_TOLERANCE,
		prevOffset: Math.max(0, Math.min(maxScroll, left - page)),
		nextOffset: Math.max(0, Math.min(maxScroll, left + page)),
		overflow
	};
}

export function jobLabel(job: string): string {
	switch (job.trim()) {
		case 'Director':
			return 'Dirección';
		case 'Writer':
		case 'Screenplay':
		case 'Novel':
		case 'Story':
			return 'Guion';
		case 'Producer':
		case 'Executive Producer':
			return 'Producción';
		case 'Original Music Composer':
			return 'Música';
		case 'Director of Photography':
			return 'Fotografía';
		case 'Editor':
			return 'Montaje';
		case 'Creator':
			return 'Creación';
		default:
			return job.trim();
	}
}

export function personInitials(name: string): string {
	const words = name
		.trim()
		.split(/\s+/)
		.filter(Boolean);
	if (words.length === 0) {
		return '';
	}
	const first = firstLetter(words[0]);
	if (words.length === 1) {
		return first;
	}
	return first + firstLetter(words[words.length - 1]);
}

function firstLetter(word: string): string {
	const normalized = word.normalize('NFD').replace(/[\u0300-\u036f]/g, '');
	return (normalized[0] ?? word[0] ?? '').toUpperCase();
}
