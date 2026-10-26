import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { ItemsResponse } from '$lib/api';
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

	try {
		const [all, recent] = await Promise.all([
			api.get<ItemsResponse>('/api/v1/items?kind=movie&limit=1', token),
			api.get<ItemsResponse>('/api/v1/items?kind=movie&sort=added&limit=12', token)
		]);
		return {
			moviesTotal: all.total,
			recent: recent.items,
			errorMessage: null
		};
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
		return {
			moviesTotal: 0,
			recent: [],
			errorMessage:
				error instanceof ConfigurationError
					? error.message
					: 'No se pudo cargar el portal. Inténtalo de nuevo en unos momentos.'
		};
	}
};
