import type { MusicAlbum, MusicAlbumSort, MusicTrack } from './api';
import type { MusicQueueTrack } from './music-queue';

export const MUSIC_VIEWS = ['albumes', 'artistas', 'pistas'] as const;
export type MusicView = (typeof MUSIC_VIEWS)[number];

export const MUSIC_ORDERS = ['title', 'artist', 'year', 'added'] as const;
export type MusicOrder = MusicAlbumSort;

export const MUSIC_VIEW_LABELS: Record<MusicView, string> = {
	albumes: 'Álbumes',
	artistas: 'Artistas',
	pistas: 'Pistas'
};

export const MUSIC_ORDER_LABELS: Record<MusicOrder, string> = {
	title: 'Título',
	artist: 'Artista',
	year: 'Año',
	added: 'Añadidos'
};

export interface MusicFilters {
	view: MusicView;
	q: string;
	letter: string | null;
	order: MusicOrder;
	page: number;
}

export function parseMusicView(raw: unknown): MusicView {
	const value = String(raw ?? '').trim().toLowerCase();
	return (MUSIC_VIEWS as readonly string[]).includes(value) ? (value as MusicView) : 'albumes';
}

export function parseMusicOrder(raw: unknown): MusicOrder {
	const value = String(raw ?? '').trim().toLowerCase();
	return (MUSIC_ORDERS as readonly string[]).includes(value) ? (value as MusicOrder) : 'title';
}

export function parseMusicLetter(raw: unknown): string | null {
	const value = String(raw ?? '').trim();
	if (value === '') {
		return null;
	}
	if (value === '#') {
		return '#';
	}
	const characters = [...value];
	if (characters.length !== 1 || !/^\p{L}$/u.test(characters[0])) {
		return null;
	}
	return characters[0].toLocaleUpperCase('es');
}

export function parseMusicPage(raw: unknown): number {
	const value = Number.parseInt(String(raw ?? '').trim(), 10);
	return Number.isInteger(value) && value >= 1 ? value : 1;
}

export function parseMusicID(raw: unknown): number | null {
	const value = Number(raw);
	return Number.isInteger(value) && value > 0 ? value : null;
}

export function readMusicFilters(url: URL): MusicFilters {
	return {
		view: parseMusicView(url.searchParams.get('vista')),
		q: (url.searchParams.get('q') ?? '').trim(),
		letter: parseMusicLetter(url.searchParams.get('letra')),
		order: parseMusicOrder(url.searchParams.get('orden')),
		page: parseMusicPage(url.searchParams.get('page'))
	};
}

export function musicApiQuery(
	filters: MusicFilters,
	page: number,
	pageSize: number
): URLSearchParams {
	const params = new URLSearchParams();
	params.set('limit', String(pageSize));
	params.set('offset', String((Math.max(1, page) - 1) * pageSize));
	if (filters.q) {
		params.set('q', filters.q);
	}
	if (filters.view !== 'pistas' && filters.letter) {
		params.set('letter', filters.letter);
	}
	if (filters.view === 'albumes') {
		params.set('sort', filters.order);
	}
	return params;
}

export function musicHref(
	currentUrl: string,
	changes: Record<string, string | null>
): string {
	const url = new URL(currentUrl, 'http://cinecat.local');
	for (const [key, value] of Object.entries(changes)) {
		if (value === null) {
			url.searchParams.delete(key);
		} else {
			url.searchParams.set(key, value);
		}
	}
	url.searchParams.delete('page');
	return `${url.pathname}${url.search}`;
}

export function musicTabHref(currentUrl: string, view: MusicView): string {
	return musicHref(currentUrl, {
		vista: view === 'albumes' ? null : view
	});
}

export function musicLetterHref(currentUrl: string, letter: string, active: boolean): string {
	return musicHref(currentUrl, { letra: active ? null : letter });
}

export function musicSearchPlaceholder(view: MusicView): string {
	if (view === 'artistas') {
		return 'Nombre del artista…';
	}
	if (view === 'pistas') {
		return 'Título, artista o álbum…';
	}
	return 'Álbum o artista…';
}

export function musicCountLabel(view: MusicView, total: number): string {
	const count = Number.isFinite(total) && total > 0 ? Math.floor(total) : 0;
	if (view === 'artistas') {
		return `${count} ${count === 1 ? 'artista' : 'artistas'}`;
	}
	if (view === 'pistas') {
		return `${count} ${count === 1 ? 'pista' : 'pistas'}`;
	}
	return `${count} ${count === 1 ? 'álbum' : 'álbumes'}`;
}

export function musicCoverURL(id: number): string {
	return `/music/cover/${id}`;
}

export function albumMeta(album: Pick<MusicAlbum, 'year' | 'track_count'>): string {
	const parts: string[] = [];
	if (album.year) {
		parts.push(String(album.year));
	}
	parts.push(`${album.track_count} ${album.track_count === 1 ? 'pista' : 'pistas'}`);
	return parts.join(' · ');
}

export function albumInitials(title: string): string {
	const words = title
		.trim()
		.split(/\s+/)
		.filter(Boolean);
	if (words.length === 0) {
		return '♪';
	}
	return words
		.slice(0, 2)
		.map((word) => [...word][0]?.toLocaleUpperCase('es') ?? '')
		.join('');
}

export function formatTrackDuration(seconds: number | null | undefined): string {
	if (seconds === null || seconds === undefined || !Number.isFinite(seconds) || seconds < 0) {
		return '—';
	}
	const total = Math.round(seconds);
	const minutes = Math.floor(total / 60);
	const rest = total % 60;
	return `${minutes}:${String(rest).padStart(2, '0')}`;
}

export function formatTotalDuration(seconds: number | null | undefined): string {
	if (seconds === null || seconds === undefined || !Number.isFinite(seconds) || seconds <= 0) {
		return '';
	}
	const total = Math.round(seconds);
	let hours = Math.floor(total / 3600);
	let minutes = Math.round((total % 3600) / 60);
	if (minutes === 60) {
		hours += 1;
		minutes = 0;
	}
	if (hours > 0) {
		return minutes > 0 ? `${hours} h ${minutes} min` : `${hours} h`;
	}
	if (minutes > 0) {
		return `${minutes} min`;
	}
	return `${total} s`;
}

export function trackPlayableReason(track: Pick<MusicTrack, 'playable'>): string {
	return track.playable ? '' : 'Formato no compatible con el navegador';
}

export function toQueueTrack(
	track: MusicTrack,
	album?: Pick<MusicAlbum, 'id' | 'title' | 'artist_id'> | null
): MusicQueueTrack {
	return {
		id: track.id,
		title: track.title,
		artist: track.artist,
		artistId: album?.artist_id,
		album: album?.title ?? track.album ?? '',
		albumId: album?.id ?? track.album_id ?? 0,
		durationSeconds: track.duration_seconds ?? 0
	};
}

export function albumQueueTracks(album: MusicAlbumDetailLike): MusicQueueTrack[] {
	return album.tracks.map((track) => toQueueTrack(track, album));
}

interface MusicAlbumDetailLike {
	id: number;
	title: string;
	artist: string;
	artist_id: number;
	tracks: MusicTrack[];
}

export function playableTracks(tracks: MusicTrack[]): MusicTrack[] {
	return tracks.filter((track) => track.playable);
}
