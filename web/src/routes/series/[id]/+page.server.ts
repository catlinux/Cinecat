import { error, fail, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { MediaItem, SeriesDetail, SeriesSeasonDetail } from '$lib/api';
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

export const load: PageServerLoad = async ({ params, locals, fetch, url }) => {
	const token = requireSessionToken(locals);
	const id = seriesID(params);
	if (id === null) {
		error(404, 'Serie no encontrada');
	}

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
		firstEpisode
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
	}
};
