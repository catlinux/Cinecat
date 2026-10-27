import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { MediaItem, PlayQuery, PlayResponse } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { sanitizeClientCapabilities } from '$lib/client-caps';
import { apiBaseUrl } from '$lib/config';
import { resolveStartAt } from '$lib/progress';
import { requireSessionToken } from '$lib/session';
import { loadEpisodeNavigation } from '$lib/episode-navigation';
import type { PageServerLoad } from './$types';

const QUALITY_VALUES = new Set(['auto', 'original', '480', '720', '1080']);

function queryOptions(url: URL, caps: string): PlayQuery {
	const query: PlayQuery = { caps };
	const audio = url.searchParams.get('audio');
	if (audio !== null && /^\d+$/.test(audio)) {
		query.audio = Number(audio);
	}
	const quality = url.searchParams.get('quality');
	if (quality !== null && QUALITY_VALUES.has(quality)) {
		query.quality = quality;
	}
	const subtitle = url.searchParams.get('subtitle');
	if (subtitle !== null && (subtitle === '' || /^[ex]\d+$/.test(subtitle))) {
		query.subtitle = subtitle;
	}
	return query;
}

export const load: PageServerLoad = async ({ params, locals, fetch, url, cookies }) => {
	const token = requireSessionToken(locals);
	const caps = sanitizeClientCapabilities(cookies?.get('cinecat_caps') ?? '');
	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		error(404, 'Ítem no encontrado');
	}

	const api = createApiClient(fetch);

	let item: MediaItem;
	try {
		item = await api.getItem(id, token);
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) {
			error(404, 'Ítem no encontrado');
		}
		if (err instanceof ApiError && err.status === 401) {
			redirect(303, '/login');
		}
		if (err instanceof ConfigurationError) {
			error(500, err.message);
		}
		error(502, 'No se pudo cargar el ítem.');
	}

	let play: PlayResponse | null = null;
	let playError: string | null = null;
	const options = queryOptions(url, caps);
	try {
		try {
			play = await api.getPlay(id, options, token);
		} catch (err) {
			// Una opción guardada obsoleta puede provocar un 400. Se reintenta
			// sin ella para no impedir la reproducción.
			if (err instanceof ApiError && err.status === 400) {
				play = await api.getPlay(id, { caps }, token);
			} else {
				throw err;
			}
		}
	} catch (err) {
		if (err instanceof ApiError) {
			if (err.status === 401) {
				redirect(303, '/login');
			}
			if (err.status === 404) {
				error(404, 'El archivo del ítem no está disponible.');
			}
			if (err.status === 415) {
				playError = 'Este contenedor todavía no se puede reproducir en Cinecat.';
			} else if (err.status === 503) {
				playError = 'El servidor no tiene FFmpeg disponible para preparar el remux.';
			}
		}
		if (!playError) {
			playError = 'No se pudo preparar la reproducción. Inténtalo de nuevo en unos momentos.';
		}
	}

	let nextEpisodeUrl: string | null = null;
	const navigation = await loadEpisodeNavigation(api, item, token);
	if (navigation?.next) {
		nextEpisodeUrl = `/items/${navigation.next.id}/play`;
	}

	const base = apiBaseUrl.replace(/\/+$/, '');
	return {
		item,
		play,
		options: play,
		nextEpisodeUrl,
		playUrl: play ? `${base}${play.url}` : null,
		playError,
		startAt: resolveStartAt(item.progress, url.searchParams.get('t'))
	};
};
