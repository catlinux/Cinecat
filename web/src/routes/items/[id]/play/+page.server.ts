import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { MediaItem, PlayResponse } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { apiBaseUrl } from '$lib/config';
import { resolveStartAt } from '$lib/progress';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, locals, fetch, url }) => {
	const token = requireSessionToken(locals);
	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		error(404, 'Ítem no encontrado');
	}

	const api = createApiClient(fetch);

	let item: MediaItem;
	try {
		item = await api.get<MediaItem>(`/api/v1/items/${id}`, token);
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
	try {
		play = await api.get<PlayResponse>(`/api/v1/items/${id}/play`, token);
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

	const base = apiBaseUrl.replace(/\/+$/, '');
	return {
		item,
		play,
		playUrl: play ? `${base}${play.url}` : null,
		playError,
		startAt: resolveStartAt(item.progress, url.searchParams.get('t'))
	};
};
