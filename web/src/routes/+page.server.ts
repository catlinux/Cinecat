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

	try {
		const home = await createApiClient(fetch).getHome(token);
		return {
			moviesTotal: home.counts.movies,
			continueWatching: home.continue ?? [],
			nextUp: home.next_up ?? [],
			recent: home.recent_movies ?? [],
			errorMessage: null
		};
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
		return {
			moviesTotal: 0,
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
