import { error, fail, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type {
	Candidate,
	FileInfoError,
	FileInfoResult,
	ImageKind,
	ImageOptions,
	MediaItem,
	PlayResponse
} from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { loadEpisodeNavigation } from '$lib/episode-navigation';
import { requireSessionToken } from '$lib/session';
import type { Actions, PageServerLoad } from './$types';

function itemID(params: { id?: string }): number | null {
	const id = Number(params.id);
	return Number.isInteger(id) && id > 0 ? id : null;
}

export const load: PageServerLoad = async ({ params, locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const id = itemID(params);
	if (id === null) {
		error(404, 'Ítem no encontrado');
	}

	const layout = typeof parent === 'function' ? await parent() : { isAdmin: false };
	const isAdmin = Boolean(layout?.isAdmin);

	try {
		const api = createApiClient(fetch);
		const item = await api.getItem(id, token);
		const isClip = item.library_kind === 'clips';
		const fileInfo: Promise<FileInfoResult> = api.getFileInfo(id, token).then(
			(data) => ({ data }),
			(cause) => ({ error: fileInfoError(cause) })
		);
		const playbackOptions: Promise<PlayResponse | null> = api
			.getPlay(id, {}, token)
			.catch(() => null);
		const similarItems: Promise<MediaItem[] | null> | null =
			item.kind === 'movie' && !isClip
				? api.getSimilarItems(id, 12, token).then((response) => response.items).catch(() => null)
				: null;
		const episodeNavigation = await loadEpisodeNavigation(api, item, token);
		return {
			item,
			isAdmin,
			corregir: isAdmin && !isClip && url.searchParams.get('corregir') === '1',
			imagenes: isAdmin && !isClip && url.searchParams.get('imagenes') === '1',
			fileInfo,
			playbackOptions,
			similarItems,
			episodeNavigation
		};
	} catch (err) {
		if (err instanceof ApiError) {
			if (err.status === 404) {
				error(404, 'Ítem no encontrado');
			}
			if (err.status === 401) {
				redirect(303, '/login');
			}
		}
		if (err instanceof ConfigurationError) {
			error(500, err.message);
		}
		error(502, 'No se pudo cargar el ítem.');
	}
};

function fileInfoError(cause: unknown): FileInfoError {
	if (cause instanceof ApiError) {
		if (cause.status === 404) {
			return 'not_found';
		}
		if (cause.status === 503) {
			return 'unavailable';
		}
	}
	return 'failed';
}

function watchedFailure(cause: unknown) {
	if (cause instanceof ApiError) {
		if (cause.status === 401) {
			redirect(303, '/login');
		}
		if (cause.status === 404) {
			return fail(404, { message: 'Ítem no encontrado.' });
		}
		if (cause.status === 403) {
			return fail(403, { message: 'No tienes permiso para cambiar el visionado.' });
		}
	}
	return fail(502, {
		message: 'No se pudo cambiar el estado de visionado. Inténtalo de nuevo.'
	});
}

function correctionFailure(cause: unknown) {
	if (cause instanceof ApiError) {
		if (cause.status === 401) {
			redirect(303, '/login');
		}
		if (cause.status === 403) {
			return fail(403, {
				panel: true,
				message: 'Solo los administradores pueden corregir fichas.'
			});
		}
		if (cause.status === 404) {
			return fail(404, { panel: true, message: 'Ítem no encontrado.' });
		}
		if (cause.status === 503) {
			return fail(503, {
				panel: true,
				message: 'TMDB no está configurado en el servidor'
			});
		}
	}
	return fail(502, { panel: true, message: 'No se pudo consultar TMDB' });
}

function imageKind(value: FormDataEntryValue | null): ImageKind | null {
	const kind = String(value ?? '');
	return kind === 'poster' || kind === 'backdrop' ? kind : null;
}

function imageFailure(cause: unknown) {
	if (cause instanceof ApiError) {
		if (cause.status === 401) {
			redirect(303, '/login');
		}
		if (cause.status === 400) {
			return fail(400, { imagePanel: true, message: 'URL no válida' });
		}
		if (cause.status === 403) {
			return fail(403, {
				imagePanel: true,
				message: 'Solo los administradores pueden cambiar las imágenes.'
			});
		}
		if (cause.status === 404) {
			return fail(404, { imagePanel: true, message: 'Ítem no encontrado.' });
		}
		if (cause.status === 409) {
			return fail(409, {
				imagePanel: true,
				message: 'Identifica primero la película para ver las imágenes de TMDB'
			});
		}
		if (cause.status === 503) {
			return fail(503, {
				imagePanel: true,
				message: 'TMDB no está configurado en el servidor'
			});
		}
	}
	return fail(502, { imagePanel: true, message: 'No se pudo consultar TMDB' });
}

export const actions: Actions = {
	preferences: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { preferencesPanel: true, message: 'Ítem no encontrado.' });
		}
		const form = await request.formData();
		const api = createApiClient(fetch);
		try {
			if (String(form.get('reset') ?? '') === '1') {
				await api.deletePreferences(id, token);
				return { success: true, preferencesPanel: true, message: 'Opciones restablecidas.' };
			}
			const changes: {
				audio_index?: number;
				subtitle_id?: string;
				quality?: string;
			} = {};
			const audioRaw = form.get('audio_index');
			if (audioRaw !== null && String(audioRaw).trim() !== '') {
				const audio = Number(audioRaw);
				if (!Number.isInteger(audio) || audio < 0) {
					return fail(400, { preferencesPanel: true, message: 'Pista de audio no válida.' });
				}
				changes.audio_index = audio;
			}
			const subtitle = form.get('subtitle_id');
			if (subtitle !== null) {
				const value = String(subtitle);
				if (value !== '' && !/^[ex]\d+$/.test(value)) {
					return fail(400, { preferencesPanel: true, message: 'Subtítulo no válido.' });
				}
				changes.subtitle_id = value;
			}
			const quality = form.get('quality');
			if (quality !== null) {
				const value = String(quality);
				if (!['auto', 'original', '480', '720', '1080'].includes(value)) {
					return fail(400, { preferencesPanel: true, message: 'Calidad no válida.' });
				}
				changes.quality = value;
			}
			await api.putPreferences(id, changes, token);
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				redirect(303, '/login');
			}
			if (err instanceof ApiError && err.status === 404) {
				return fail(404, { preferencesPanel: true, message: 'Ítem no encontrado.' });
			}
			return fail(502, {
				preferencesPanel: true,
				message: 'No se pudieron guardar las opciones de reproducción.'
			});
		}
		return { success: true, preferencesPanel: true, message: 'Opciones guardadas.' };
	},

	watched: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { message: 'Ítem no encontrado.' });
		}

		const form = await request.formData();
		const requested = String(form.get('watched') ?? '');
		if (requested !== '1' && requested !== '0') {
			return fail(400, { message: 'Estado de visionado no válido.' });
		}

		try {
			await createApiClient(fetch).setWatched(id, requested === '1', token);
		} catch (err) {
			return watchedFailure(err);
		}

		return {
			success: true,
			message: requested === '1' ? 'Marcada como vista.' : 'Desmarcada.'
		};
	},

	candidates: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Ítem no encontrado.' });
		}

		const form = await request.formData();
		const query = String(form.get('q') ?? '').trim();
		if (!query) {
			return fail(400, { panel: true, message: 'Escribe un título para buscar.' });
		}

		const yearRaw = String(form.get('year') ?? '').trim();
		let year = 0;
		if (yearRaw) {
			year = Number(yearRaw);
			if (!Number.isInteger(year) || year < 0) {
				return fail(400, { panel: true, message: 'El año debe ser un número entero.' });
			}
		}

		try {
			const response = await createApiClient(fetch).getCandidates(id, query, year, token);
			return {
				panel: true,
				candidates: (response.candidates ?? []) as Candidate[],
				search: { q: query, year: year > 0 ? year : null }
			};
		} catch (err) {
			return correctionFailure(err);
		}
	},

	match: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Ítem no encontrado.' });
		}

		const form = await request.formData();
		const tmdbID = Number(String(form.get('tmdb_id') ?? ''));
		if (!Number.isInteger(tmdbID) || tmdbID <= 0) {
			return fail(400, { panel: true, message: 'Candidato no válido.' });
		}

		try {
			await createApiClient(fetch).matchItem(id, tmdbID, token);
		} catch (err) {
			return correctionFailure(err);
		}

		return { success: true, panel: true, message: 'Ficha corregida.' };
	},

	nomatch: async ({ locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Ítem no encontrado.' });
		}

		try {
			await createApiClient(fetch).matchItem(id, null, token);
		} catch (err) {
			return correctionFailure(err);
		}

		return { success: true, panel: true, message: 'Marcada como «no es ninguna».' };
	},

	automatic: async ({ locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Ítem no encontrado.' });
		}

		try {
			await createApiClient(fetch).resetMatch(id, token);
		} catch (err) {
			return correctionFailure(err);
		}

		return {
			success: true,
			panel: true,
			message: 'Vuelta a la identificación automática.'
		};
	},

	imageList: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { imagePanel: true, message: 'Ítem no encontrado.' });
		}

		const form = await request.formData();
		const kind = imageKind(form.get('kind'));
		if (kind === null) {
			return fail(400, { imagePanel: true, message: 'Tipo de imagen no válido.' });
		}

		try {
			const imageOptions: ImageOptions = await createApiClient(fetch).getImageOptions(id, token);
			return { imagePanel: true, imageKind: kind, imageOptions };
		} catch (err) {
			return imageFailure(err);
		}
	},

	imageSet: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { imagePanel: true, message: 'Ítem no encontrado.' });
		}

		const form = await request.formData();
		const kind = imageKind(form.get('kind'));
		if (kind === null) {
			return fail(400, { imagePanel: true, message: 'Tipo de imagen no válido.' });
		}
		const url = String(form.get('url') ?? '').trim();
		if (!url.startsWith('http://') && !url.startsWith('https://')) {
			return fail(400, { imagePanel: true, message: 'URL no válida' });
		}

		try {
			await createApiClient(fetch).setImage(id, kind, url, token);
		} catch (err) {
			return imageFailure(err);
		}

		return {
			success: true,
			imagePanel: true,
			message: kind === 'poster' ? 'Cartel actualizado.' : 'Fondo actualizado.'
		};
	},

	imageReset: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = itemID(params);
		if (id === null) {
			return fail(404, { imagePanel: true, message: 'Ítem no encontrado.' });
		}

		const form = await request.formData();
		const kind = imageKind(form.get('kind'));
		if (kind === null) {
			return fail(400, { imagePanel: true, message: 'Tipo de imagen no válido.' });
		}

		try {
			await createApiClient(fetch).resetImage(id, kind, token);
		} catch (err) {
			return imageFailure(err);
		}

		return {
			success: true,
			imagePanel: true,
			message: kind === 'poster' ? 'Cartel restaurado.' : 'Fondo restaurado.'
		};
	}
};
