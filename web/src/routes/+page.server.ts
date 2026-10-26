import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

const LEGACY_CARTELERA_PARAMS = ['q', 'page', 'letter', 'genre', 'sort'];

export const load: PageServerLoad = async ({ locals, fetch, url }) => {
	if (LEGACY_CARTELERA_PARAMS.some((param) => url.searchParams.has(param))) {
		redirect(308, `/movies${url.search}`);
	}

	const token = requireSessionToken(locals);
	const api = createApiClient(fetch);

	let collectionsTotal: number | null = null;
	try {
		const collections = await api.getCollections({ limit: 1 }, token);
		collectionsTotal = collections.total;
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
	}

	try {
		const home = await api.getHome(token);
		let seriesTotal: number | null = null;
		if (typeof home.counts.series === 'number' && Number.isFinite(home.counts.series)) {
			seriesTotal = home.counts.series;
		} else {
			try {
				const series = await api.getSeries({ limit: 1 }, token);
				seriesTotal = series.total;
			} catch (seriesError) {
				if (seriesError instanceof ApiError && seriesError.status === 401) {
					redirect(303, '/login');
				}
			}
		}
		return {
			moviesTotal: home.counts.movies,
			collectionsTotal,
			seriesTotal,
			continueWatching: home.continue ?? [],
			nextUp: home.next_up ?? [],
			recent: home.recent_movies ?? [],
			errorMessage: null
		};
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
		let seriesTotal: number | null = null;
		try {
			const series = await api.getSeries({ limit: 1 }, token);
			seriesTotal = series.total;
		} catch (seriesError) {
			if (seriesError instanceof ApiError && seriesError.status === 401) {
				redirect(303, '/login');
			}
		}
		return {
			moviesTotal: 0,
			collectionsTotal,
			seriesTotal,
			continueWatching: [],
			nextUp: [],
			recent: [],
			errorMessage:
				error instanceof ConfigurationError
					? error.message
					: 'No se pudo cargar el portal. Inténtalo de nuevo en unos momentos.'
		};
	}
};
