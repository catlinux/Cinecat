import { error, fail, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type {
	Candidate,
	ImageKind,
	ImageOptions,
	MediaItem,
	SeriesDetail,
	SeriesSeasonDetail
} from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { selectSeriesSeason } from '$lib/series';
import { requireSessionToken } from '$lib/session';
import type { Actions, PageServerLoad } from './$types';

function seriesID(params: { id?: string }): number | null {
	const id = Number(params.id);
	return Number.isInteger(id) && id > 0 ? id : null;
}

function watchedValue(value: FormDataEntryValue | null): boolean | null {
	const raw = String(value ?? '');
	if (raw === '1') {
		return true;
	}
	if (raw === '0') {
		return false;
	}
	return null;
}

function watchedFailure(cause: unknown) {
	if (cause instanceof ApiError) {
		if (cause.status === 401) {
			redirect(303, '/login');
		}
		if (cause.status === 403) {
			return fail(403, { message: 'No tienes permiso para cambiar el visionado.' });
		}
		if (cause.status === 404) {
			return fail(404, { message: 'Episodio o serie no encontrados.' });
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
		if (cause.status === 400) {
			return fail(400, { panel: true, message: 'Datos no válidos.' });
		}
		if (cause.status === 403) {
			return fail(403, {
				panel: true,
				message: 'Solo los administradores pueden corregir fichas.'
			});
		}
		if (cause.status === 404) {
			return fail(404, { panel: true, message: 'Serie no encontrada.' });
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
			return fail(404, { imagePanel: true, message: 'Serie no encontrada.' });
		}
		if (cause.status === 409) {
			return fail(409, {
				imagePanel: true,
				message: 'Identifica primero la serie para ver las imágenes de TMDB'
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

export const load: PageServerLoad = async ({ params, locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const id = seriesID(params);
	if (id === null) {
		error(404, 'Serie no encontrada');
	}

	const layout = typeof parent === 'function' ? await parent() : { isAdmin: false };
	const isAdmin = Boolean(layout?.isAdmin);

	const api = createApiClient(fetch);
	let series: SeriesDetail;
	try {
		series = await api.getSeriesDetail(id, token);
	} catch (cause) {
		if (cause instanceof ApiError) {
			if (cause.status === 404) {
				error(404, 'Serie no encontrada');
			}
			if (cause.status === 401) {
				redirect(303, '/login');
			}
		}
		if (cause instanceof ConfigurationError) {
			error(500, cause.message);
		}
		error(502, 'No se pudo cargar la serie.');
	}

	const seasons = series.seasons ?? [];
	const selectedSeason = selectSeriesSeason(
		seasons,
		series.next_episode,
		url.searchParams.get('season')
	);

	let season: SeriesSeasonDetail | null = null;
	let firstEpisode: MediaItem | null = null;
	if (selectedSeason !== null) {
		try {
			season = await api.getSeriesSeason(id, selectedSeason, token);
		} catch (cause) {
			if (cause instanceof ApiError) {
				if (cause.status === 401) {
					redirect(303, '/login');
				}
				if (cause.status === 404) {
					error(404, 'Temporada no encontrada');
				}
			}
			error(502, 'No se pudo cargar la temporada.');
		}
	}

	// Si no hay un siguiente episodio (por ejemplo, porque ya está toda vista),
	// se necesita el primero para poder ofrecer «Volver a ver».
	const allWatched =
		series.episode_count > 0 && series.watched_count >= series.episode_count;
	if ((series.next_episode === null || allWatched) && series.episode_count > 0 && seasons.length > 0) {
		const firstSeasonNumber = seasons[0].season;
		if (selectedSeason === firstSeasonNumber && season !== null) {
			firstEpisode = season.items[0] ?? null;
		} else {
			try {
				const firstSeason = await api.getSeriesSeason(id, firstSeasonNumber, token);
				firstEpisode = firstSeason.items[0] ?? null;
			} catch (cause) {
				if (cause instanceof ApiError && cause.status === 401) {
					redirect(303, '/login');
				}
			}
		}
	}

	return {
		series,
		season,
		selectedSeason,
		firstEpisode,
		isAdmin,
		corregir: isAdmin && url.searchParams.get('corregir') === '1',
		imagenes: isAdmin && url.searchParams.get('imagenes') === '1'
	};
};

export const actions: Actions = {
	watchedSeries: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { message: 'Serie no encontrada.' });
		}
		const form = await request.formData();
		const watched = watchedValue(form.get('watched'));
		if (watched === null) {
			return fail(400, { message: 'Estado de visionado no válido.' });
		}
		try {
			await createApiClient(fetch).setSeriesWatched(id, watched, token);
		} catch (cause) {
			return watchedFailure(cause);
		}
		return {
			success: true,
			message: watched ? 'Serie marcada como vista.' : 'Serie marcada como no vista.'
		};
	},

	watchedSeason: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { message: 'Serie no encontrada.' });
		}
		const form = await request.formData();
		const seasonNumber = Number(String(form.get('season') ?? ''));
		const watched = watchedValue(form.get('watched'));
		if (!Number.isInteger(seasonNumber) || seasonNumber <= 0 || watched === null) {
			return fail(400, { message: 'Estado de visionado no válido.' });
		}
		try {
			await createApiClient(fetch).setSeriesSeasonWatched(
				id,
				seasonNumber,
				watched,
				token
			);
		} catch (cause) {
			return watchedFailure(cause);
		}
		return {
			success: true,
			message: watched
				? 'Temporada marcada como vista.'
				: 'Temporada marcada como no vista.'
		};
	},

	watchedEpisode: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const itemID = Number(String(form.get('itemId') ?? ''));
		const watched = watchedValue(form.get('watched'));
		if (!Number.isInteger(itemID) || itemID <= 0 || watched === null) {
			return fail(400, { message: 'Estado de visionado no válido.' });
		}
		try {
			await createApiClient(fetch).setWatched(itemID, watched, token);
		} catch (cause) {
			return watchedFailure(cause);
		}
		return {
			success: true,
			message: watched ? 'Episodio marcado como visto.' : 'Episodio desmarcado.'
		};
	},

	candidates: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Serie no encontrada.' });
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
			const response = await createApiClient(fetch).getSeriesCandidates(
				id,
				query,
				year,
				token
			);
			return {
				panel: true,
				candidates: (response.candidates ?? []) as Candidate[],
				search: { q: query, year: year > 0 ? year : null }
			};
		} catch (cause) {
			return correctionFailure(cause);
		}
	},

	match: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Serie no encontrada.' });
		}

		const form = await request.formData();
		const tmdbID = Number(String(form.get('tmdb_id') ?? ''));
		if (!Number.isInteger(tmdbID) || tmdbID <= 0) {
			return fail(400, { panel: true, message: 'Candidato no válido.' });
		}

		try {
			await createApiClient(fetch).matchSeries(id, tmdbID, token);
		} catch (cause) {
			return correctionFailure(cause);
		}

		return { success: true, panel: true, message: 'Ficha corregida.' };
	},

	nomatch: async ({ locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Serie no encontrada.' });
		}

		try {
			await createApiClient(fetch).matchSeries(id, null, token);
		} catch (cause) {
			return correctionFailure(cause);
		}

		return { success: true, panel: true, message: 'Marcada como «no es ninguna».' };
	},

	automatic: async ({ locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { panel: true, message: 'Serie no encontrada.' });
		}

		try {
			await createApiClient(fetch).resetSeriesMatch(id, token);
		} catch (cause) {
			return correctionFailure(cause);
		}

		return {
			success: true,
			panel: true,
			message: 'Vuelta a la identificación automática.'
		};
	},

	imageList: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { imagePanel: true, message: 'Serie no encontrada.' });
		}

		const form = await request.formData();
		const kind = imageKind(form.get('kind'));
		if (kind === null) {
			return fail(400, { imagePanel: true, message: 'Tipo de imagen no válido.' });
		}

		try {
			const imageOptions: ImageOptions = await createApiClient(fetch).getSeriesImageOptions(
				id,
				token
			);
			return { imagePanel: true, imageKind: kind, imageOptions };
		} catch (cause) {
			return imageFailure(cause);
		}
	},

	imageSet: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { imagePanel: true, message: 'Serie no encontrada.' });
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
			await createApiClient(fetch).setSeriesImage(id, kind, url, token);
		} catch (cause) {
			return imageFailure(cause);
		}

		return {
			success: true,
			imagePanel: true,
			message: kind === 'poster' ? 'Cartel actualizado.' : 'Fondo actualizado.'
		};
	},

	imageReset: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = seriesID(params);
		if (id === null) {
			return fail(404, { imagePanel: true, message: 'Serie no encontrada.' });
		}

		const form = await request.formData();
		const kind = imageKind(form.get('kind'));
		if (kind === null) {
			return fail(400, { imagePanel: true, message: 'Tipo de imagen no válido.' });
		}

		try {
			await createApiClient(fetch).resetSeriesImage(id, kind, token);
		} catch (cause) {
			return imageFailure(cause);
		}

		return {
			success: true,
			imagePanel: true,
			message: kind === 'poster' ? 'Cartel restaurado.' : 'Fondo restaurado.'
		};
	}
};
