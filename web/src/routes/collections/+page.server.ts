import { fail, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { CollectionsResponse } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import {
	COLLECTIONS_PAGE_SIZE,
	collectionApiQuery,
	readCollectionFilters
} from '$lib/collections';
import { clampPage, pageCount, pageHref } from '$lib/pagination';
import { requireSessionToken } from '$lib/session';
import type { Actions, PageServerLoad } from './$types';

const ERROR_MESSAGE = 'No se pudieron cargar las colecciones. Inténtalo de nuevo en unos momentos.';

export const load: PageServerLoad = async ({ locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const layout = typeof parent === 'function' ? await parent() : { isAdmin: false };
	const isAdmin = Boolean(layout?.isAdmin);
	const filters = readCollectionFilters(url);
	const currentUrl = `${url.pathname}${url.search}`;
	const requestedPage = filters.page;

	let response: CollectionsResponse;
	try {
		response = await createApiClient(fetch).get<CollectionsResponse>(
			`/api/v1/collections?${collectionApiQuery(filters, requestedPage)}`,
			token
		);
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
		return {
			collections: [],
			total: 0,
			page: 1,
			last: 1,
			pageSize: COLLECTIONS_PAGE_SIZE,
			filters,
			currentUrl,
			isAdmin,
			errorMessage: error instanceof ApiError ? error.message : ERROR_MESSAGE
		};
	}

	const last = pageCount(response.total, COLLECTIONS_PAGE_SIZE);
	const page = clampPage(requestedPage, last);
	if (page !== requestedPage) {
		redirect(303, pageHref(url, page));
	}

	return {
		collections: response.collections,
		total: response.total,
		page,
		last,
		pageSize: COLLECTIONS_PAGE_SIZE,
		filters,
		currentUrl,
		isAdmin,
		errorMessage: null
	};
};

export const actions: Actions = {
	create: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const overview = String(form.get('overview') ?? '').trim();
		if (Array.from(name).length < 1 || Array.from(name).length > 120) {
			return fail(400, { message: 'El nombre debe tener entre 1 y 120 caracteres.' });
		}
		if (Array.from(overview).length > 2000) {
			return fail(400, { message: 'La descripción no puede superar los 2000 caracteres.' });
		}

		let collectionID: number;
		try {
			const collection = await createApiClient(fetch).createCollection(name, overview, token);
			collectionID = collection.id;
		} catch (error) {
			if (error instanceof ApiError) {
				if (error.status === 401) {
					redirect(303, '/login');
				}
				if (error.status === 400) {
					return fail(400, { message: error.message || 'Datos no válidos.' });
				}
				if (error.status === 403) {
					return fail(403, { message: 'Solo los administradores pueden crear colecciones.' });
				}
			}
			return fail(502, { message: 'No se pudo crear la colección.' });
		}
		redirect(303, `/collections/${collectionID}`);
	}
};
