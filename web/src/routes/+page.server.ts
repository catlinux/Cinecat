import { ApiError } from '$lib/api';
import type { ItemsResponse } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

const PAGE_SIZE = 50;

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const token = requireSessionToken(locals);

	try {
		const data = await createApiClient(fetch).get<ItemsResponse>(
			`/api/v1/items?limit=${PAGE_SIZE}`,
			token
		);
		return { items: data.items, total: data.total, errorMessage: null };
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
		if (error instanceof ConfigurationError) {
			return { items: [], total: 0, errorMessage: error.message };
		}
		return {
			items: [],
			total: 0,
			errorMessage: 'No se pudo cargar la biblioteca. Inténtalo de nuevo en unos momentos.'
		};
	}
};
