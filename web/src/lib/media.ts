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
