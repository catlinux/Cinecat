import type { MediaItem } from './api';

export function kindLabel(kind: MediaItem['kind']): string {
	switch (kind) {
		case 'movie':
			return 'Película';
		case 'series':
			return 'Serie';
		case 'episode':
			return 'Episodio';
		default:
			return kind;
	}
}

// Texto corto de apoyo: año y, en episodios, temporada y capítulo.
export function itemSubtitle(item: MediaItem): string {
	const parts: string[] = [];

	if (item.kind === 'episode' && item.season !== null && item.episode !== null) {
		parts.push(`T${item.season}E${item.episode}`);
	} else if (item.kind === 'episode' && item.episode !== null) {
		parts.push(`Episodio ${item.episode}`);
	}

	if (item.year !== null) {
		parts.push(String(item.year));
	}

	return parts.length > 0 ? parts.join(' · ') : 'Sin año';
}

export function formatRuntime(minutes: number | null): string {
	if (minutes === null || !Number.isFinite(minutes) || minutes <= 0) {
		return '';
	}
	const total = Math.round(minutes);
	const hours = Math.floor(total / 60);
	const rest = total % 60;
	if (hours === 0) {
		return `${rest} min`;
	}
	if (rest === 0) {
		return `${hours} h`;
	}
	return `${hours} h ${rest} min`;
}

// La nota se muestra siempre con una cifra decimal y coma castellana.
export function formatRating(rating: number | null): string {
	if (rating === null || !Number.isFinite(rating)) {
		return '';
	}
	return `★ ${rating.toFixed(1).replace('.', ',')}`;
}

export function formatGenres(genres: string[]): string {
	return genres
		.map((genre) => genre.trim())
		.filter(Boolean)
		.join(', ');
}

export function formatMovieMeta(
	item: Pick<MediaItem, 'year' | 'runtime_minutes' | 'genres' | 'rating'>
): string {
	const parts: string[] = [];
	if (item.year !== null && item.year > 0) {
		parts.push(String(item.year));
	}
	const runtime = formatRuntime(item.runtime_minutes);
	if (runtime) {
		parts.push(runtime);
	}
	const genres = formatGenres(item.genres);
	if (genres) {
		parts.push(genres);
	}
	const rating = formatRating(item.rating);
	if (rating) {
		parts.push(rating);
	}
	return parts.join(' · ');
}

export function splitCast(cast: string[], limit = 12): { visible: string[]; hidden: string[] } {
	const names = cast.map((name) => name.trim()).filter(Boolean);
	return {
		visible: names.slice(0, limit),
		hidden: names.slice(limit)
	};
}

export function overviewInSpanish(language: string | null): boolean {
	return (language ?? '').trim().toLowerCase().startsWith('es');
}

export function matchStatusLabel(
	item: Pick<MediaItem, 'match_status' | 'match_score'>
): string {
	switch (item.match_status) {
		case 'auto':
			if (item.match_score === null || !Number.isFinite(item.match_score)) {
				return 'Identificada automáticamente';
			}
			return `Identificada automáticamente (${Math.round(item.match_score * 100)} %)`;
		case 'manual':
			return 'Corregida a mano';
		case 'none':
			return 'Marcada como «no es ninguna»';
		default:
			return 'Sin identificar';
	}
}

export function truncateSynopsis(text: string, maxLength = 140): string {
	const clean = text.replace(/\s+/g, ' ').trim();
	if (clean.length <= maxLength) {
		return clean;
	}
	const cut = clean.slice(0, Math.max(0, maxLength - 1));
	const lastSpace = cut.lastIndexOf(' ');
	const base = lastSpace > maxLength / 2 ? cut.slice(0, lastSpace) : cut;
	return `${base.trimEnd()}…`;
}

export function formatExpiry(value: string): string {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return value;
	}
	return new Intl.DateTimeFormat('es-ES', {
		dateStyle: 'short',
		timeStyle: 'short'
	}).format(date);
}
