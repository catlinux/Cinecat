import type { MediaItem, PlaybackAudioTrack, PlaybackSubtitle, SeriesSeason } from './api';

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

export function episodeImageURL(item: MediaItem): string | null {
	return (
		item.still_url ||
		item.series_backdrop_url ||
		item.series_poster_url ||
		item.poster_url ||
		null
	);
}

export function episodeCardSubtitle(item: MediaItem): string {
	const parts: string[] = [];
	if (item.series_title?.trim()) {
		parts.push(item.series_title.trim());
	}
	if (item.season !== null && item.episode !== null) {
		parts.push(`T${item.season} E${item.episode}`);
	}
	return parts.join(' · ');
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

function formatDecimal(value: number, decimals: number): string {
	let text = value.toFixed(decimals);
	if (text.includes('.')) {
		text = text.replace(/0+$/, '').replace(/\.$/, '');
	}
	return text.replace('.', ',');
}

export function formatDurationSeconds(seconds: number | null): string {
	if (seconds === null || !Number.isFinite(seconds) || seconds <= 0) {
		return '';
	}
	const total = Math.round(seconds);
	const hours = Math.floor(total / 3600);
	const minutes = Math.floor((total % 3600) / 60);
	const rest = total % 60;
	if (hours > 0) {
		return minutes > 0 ? `${hours} h ${minutes} min` : `${hours} h`;
	}
	if (minutes > 0) {
		return `${minutes} min`;
	}
	return `${rest} s`;
}

// Los tamaños se muestran en unidades de 1024 bytes, como en el sistema de archivos.
export function formatBytes(bytes: number | null): string {
	if (bytes === null || !Number.isFinite(bytes) || bytes < 0) {
		return '—';
	}
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let value = bytes;
	let unit = 0;
	while (value >= 1024 && unit < units.length - 1) {
		value /= 1024;
		unit++;
	}
	const decimals = unit === 0 || value >= 100 ? 0 : 1;
	return `${formatDecimal(value, decimals)} ${units[unit]}`;
}

export function formatBitrate(bitsPerSecond: number | null, forceMbps = false): string {
	if (bitsPerSecond === null || !Number.isFinite(bitsPerSecond) || bitsPerSecond <= 0) {
		return '—';
	}
	if (forceMbps || bitsPerSecond >= 1_000_000) {
		return `${formatDecimal(bitsPerSecond / 1_000_000, 1)} Mb/s`;
	}
	if (bitsPerSecond >= 1000) {
		return `${formatDecimal(bitsPerSecond / 1000, 0)} kb/s`;
	}
	return `${formatDecimal(bitsPerSecond, 0)} b/s`;
}

export function formatAudioChannels(channels: number, layout: string | null): string {
	const clean = (layout ?? '')
		.toLowerCase()
		.replace(/\s*\([^)]*\)\s*/g, '')
		.trim();
	if (channels === 1 || clean === 'mono') {
		return 'mono';
	}
	if (channels === 2 || clean === 'stereo') {
		return 'estéreo';
	}
	if (clean === '5.1' || channels === 6) {
		return '5.1';
	}
	if (clean === '7.1' || channels === 8) {
		return '7.1';
	}
	if (clean !== '') {
		return clean;
	}
	return channels > 0 ? `${channels} canales` : '';
}

export function formatAudioCodec(codec: string): string {
	switch (codec.toLowerCase()) {
		case 'ac3':
			return 'AC-3';
		case 'eac3':
			return 'E-AC-3';
		case 'truehd':
			return 'TrueHD';
		case 'dts':
			return 'DTS';
		case 'aac':
			return 'AAC';
		default:
			return codec.toUpperCase();
	}
}

export function formatAudioTrack(track: PlaybackAudioTrack): string {
	const parts = [track.language_name || 'Desconocido', formatAudioCodec(track.codec)];
	if (track.channel_layout) {
		parts.push(track.channel_layout);
	} else if (track.channels > 0) {
		parts.push(`${track.channels} canales`);
	}
	if (track.title) {
		parts.push(track.title);
	}
	return parts.join(' · ');
}

export function formatSubtitleTrack(track: PlaybackSubtitle): string {
	const parts = [track.language_name || 'Desconocido'];
	if (track.title) {
		parts.push(track.title);
	}
	if (track.forced) {
		parts.push('Forzados');
	}
	return parts.join(' · ');
}

export function formatLanguage(code: string | null): string {
	const original = (code ?? '').trim();
	if (!original) {
		return '';
	}
	const base = original.toLowerCase().split(/[-_]/)[0];
	const names: Record<string, string> = {
		es: 'Español',
		spa: 'Español',
		en: 'Inglés',
		eng: 'Inglés',
		fr: 'Francés',
		fre: 'Francés',
		fra: 'Francés',
		de: 'Alemán',
		ger: 'Alemán',
		deu: 'Alemán',
		it: 'Italiano',
		ita: 'Italiano',
		pt: 'Portugués',
		por: 'Portugués',
		ca: 'Catalán',
		cat: 'Catalán',
		ja: 'Japonés',
		jpn: 'Japonés'
	};
	return names[base] ?? original;
}

export function formatFps(rate: number | null): string {
	if (rate === null || !Number.isFinite(rate) || rate <= 0) {
		return '—';
	}
	return formatDecimal(rate, 3);
}

export function formatResolution(
	width: number,
	height: number,
	displayAspectRatio: string | null
): string {
	if (width <= 0 || height <= 0) {
		return '—';
	}
	const base = `${width} × ${height}`;
	const aspect = (displayAspectRatio ?? '').trim();
	return aspect ? `${base} (${aspect})` : base;
}

export function formatSampleRate(hertz: number | null): string {
	if (hertz === null || !Number.isFinite(hertz) || hertz <= 0) {
		return '—';
	}
	if (hertz >= 1000) {
		return `${formatDecimal(hertz / 1000, 1)} kHz`;
	}
	return `${formatDecimal(hertz, 0)} Hz`;
}

export function formatDateTime(value: string | null): string {
	if (!value) {
		return '';
	}
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return value;
	}
	return new Intl.DateTimeFormat('es-ES', {
		dateStyle: 'medium',
		timeStyle: 'short'
	}).format(date);
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

export interface EpisodeNeighbors {
	previous: MediaItem | null;
	next: MediaItem | null;
}

function sortEpisodes(items: MediaItem[]): MediaItem[] {
	return [...items].sort((a, b) => {
		const left = a.episode ?? Number.MAX_SAFE_INTEGER;
		const right = b.episode ?? Number.MAX_SAFE_INTEGER;
		return left - right || a.id - b.id;
	});
}

// La temporada pedida puede no ser la anterior numérica: la lista de temporadas
// manda para saltar huecos sin inventarse una temporada que no existe.
export function episodeNeighbors(
	current: Pick<MediaItem, 'id' | 'season' | 'episode'>,
	seasons: Array<Pick<SeriesSeason, 'season'>>,
	itemsBySeason: Map<number, MediaItem[]>
): EpisodeNeighbors {
	if (current.season === null) {
		return { previous: null, next: null };
	}

	const seasonNumbers = [...new Set(seasons.map((season) => season.season))].sort(
		(left, right) => left - right
	);
	const seasonIndex = seasonNumbers.indexOf(current.season);
	if (seasonIndex === -1) {
		return { previous: null, next: null };
	}

	const currentItems = sortEpisodes(itemsBySeason.get(current.season) ?? []);
	const currentIndex = currentItems.findIndex((item) => item.id === current.id);
	if (currentIndex === -1) {
		return { previous: null, next: null };
	}

	let previous: MediaItem | null = currentIndex > 0 ? currentItems[currentIndex - 1] : null;
	for (let index = seasonIndex - 1; previous === null && index >= 0; index -= 1) {
		const items = sortEpisodes(itemsBySeason.get(seasonNumbers[index]) ?? []);
		if (items.length > 0) {
			previous = items[items.length - 1];
		}
	}

	let next: MediaItem | null =
		currentIndex < currentItems.length - 1 ? currentItems[currentIndex + 1] : null;
	for (let index = seasonIndex + 1; next === null && index < seasonNumbers.length; index += 1) {
		const items = sortEpisodes(itemsBySeason.get(seasonNumbers[index]) ?? []);
		if (items.length > 0) {
			next = items[0];
		}
	}

	return { previous, next };
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
