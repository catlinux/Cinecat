import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { MediaItem, PlayResponse } from '$lib/api';
import PlaybackOptions from '$lib/components/PlaybackOptions.svelte';
import ItemPage from './+page.svelte';
import { actions, load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const item: MediaItem = {
	id: 7,
	library_id: 1,
	path: '/media/El Padrino.mkv',
	title: 'El Padrino',
	file_title: 'El Padrino',
	year: 1972,
	kind: 'movie',
	season: null,
	episode: null,
	tmdb_id: 238,
	overview: null,
	poster_url: null,
	poster_custom: false,
	original_title: null,
	imdb_id: null,
	tagline: null,
	genres: [],
	runtime_minutes: null,
	rating: null,
	directors: [],
	cast: [],
	backdrop_url: null,
	backdrop_custom: false,
	overview_language: null,
	match_status: 'auto',
	match_score: 1,
	watched: false,
	progress: null,
	collections: [],
	series_id: null,
	series_title: null,
	episode_title: null,
	still_url: null
};

const play: PlayResponse = {
	url: '/api/v1/stream/7?exp=1&sig=abc',
	mode: 'remux',
	expires_at: '2026-10-26T12:00:00Z',
	audio_tracks: [
		{
			index: 0,
			language: 'spa',
			language_name: 'Español',
			codec: 'ac3',
			channels: 6,
			channel_layout: '5.1',
			title: '',
			default: true
		},
		{
			index: 1,
			language: 'eng',
			language_name: 'Inglés',
			codec: 'aac',
			channels: 2,
			channel_layout: 'stereo',
			title: '',
			default: false
		}
	],
	selected_audio: 1,
	qualities: [
		{ value: 'auto', label: 'Auto (hasta 1080p)' },
		{ value: '720', label: '720p' },
		{ value: 'original', label: 'Original' }
	],
	selected_quality: '720',
	subtitles: [
		{
			id: 'e0',
			source: 'embedded',
			language: 'spa',
			language_name: 'Español',
			title: '',
			codec: 'subrip',
			forced: false,
			default: false,
			url: '/subtitles/e0.vtt'
		}
	],
	selected_subtitle: 'e0'
};

function renderOptions(): string {
	return render(PlaybackOptions, { props: { options: play } }).body;
}

function loadEvent(fetchImpl: typeof fetch) {
	return {
		params: { id: '7' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL('http://localhost/items/7'),
		parent: async () => ({ isAdmin: false })
	};
}

function actionEvent(fetchImpl: typeof fetch, fields: Record<string, string>) {
	return {
		params: { id: '7' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL('http://localhost/items/7'),
		request: new Request('http://localhost/items/7?/preferences', {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		})
	};
}

describe('selector de reproducción de la ficha', () => {
	it('muestra los tres selectores con la selección guardada', () => {
		const body = renderOptions();
		expect(body).toContain('name="audio_index"');
		expect(body).toContain('name="subtitle_id"');
		expect(body).toContain('name="quality"');
		expect(body).toContain('Español · AC-3 · 5.1');
		expect(body).toContain('Inglés · AAC · stereo');
		expect(body).toContain('Auto (hasta 1080p)');
		expect(body).toContain('action="?/preferences"');
		expect(body).toContain('Restablecer');
		expect(body).toContain('selected');
	});

	it('la ficha muestra el bloque con la respuesta de /play', () => {
		const body = render(ItemPage, {
			props: {
				data: {
					item,
					isAdmin: false,
					corregir: false,
					playbackOptions: play
				}
			} as never
		}).body;
		expect(body).toContain('Reproducción');
		expect(body).toContain('name="audio_index"');
		expect(body).toContain('name="quality"');
	});

	it('guarda las opciones con PUT', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const result = (await actions.preferences!(
			actionEvent(fetchMock, {
				audio_index: '1',
				subtitle_id: 'e0',
				quality: '720'
			}) as never
		)) as { success?: boolean };
		expect(result.success).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/preferences');
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body)).toEqual({
			audio_index: 1,
			subtitle_id: 'e0',
			quality: '720'
		});
	});

	it('restablece con DELETE', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		await actions.preferences!(actionEvent(fetchMock, { reset: '1' }) as never);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/preferences');
		expect(init.method).toBe('DELETE');
	});

	it('oculta el bloque si /play falla', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(item);
			}
			return jsonResponse({ error: 'sin ffmpeg' }, 503);
		});
		const result = (await load(loadEvent(fetchMock) as never)) as {
			playbackOptions: Promise<PlayResponse | null>;
		};
		await expect(result.playbackOptions).resolves.toBeNull();
	});
});
