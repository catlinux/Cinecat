import { error, fail, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { CollectionDetail, ItemsResponse } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { moveItem } from '$lib/collections';
import { requireSessionToken } from '$lib/session';
import type { Actions, PageServerLoad } from './$types';

function collectionID(params: { id?: string }): number | null {
	const id = Number(params.id);
	return Number.isInteger(id) && id > 0 ? id : null;
}

function manageFailure(cause: unknown) {
	if (cause instanceof ApiError) {
		if (cause.status === 401) {
			redirect(303, '/login');
		}
		if (cause.status === 400) {
			return fail(400, { manage: true, message: cause.message || 'Datos no válidos.' });
		}
		if (cause.status === 403) {
			return fail(403, {
				manage: true,
				message: 'Solo los administradores pueden gestionar colecciones.'
			});
		}
		if (cause.status === 404) {
			return fail(404, { manage: true, message: 'Colección no encontrada.' });
		}
	}
	return fail(502, { manage: true, message: 'No se pudo completar la operación.' });
}

export const load: PageServerLoad = async ({ params, locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const id = collectionID(params);
	if (id === null) {
		error(404, 'Colección no encontrada');
	}
	const layout = typeof parent === 'function' ? await parent() : { isAdmin: false };
	const isAdmin = Boolean(layout?.isAdmin);

	let collection: CollectionDetail;
	try {
		collection = await createApiClient(fetch).getCollection(id, token);
	} catch (cause) {
		if (cause instanceof ApiError) {
			if (cause.status === 404) {
				error(404, 'Colección no encontrada');
			}
			if (cause.status === 401) {
				redirect(303, '/login');
			}
		}
		error(502, 'No se pudo cargar la colección.');
	}

	const searchQuery = isAdmin ? (url.searchParams.get('buscar') ?? '').trim() : '';
	let searchResults: ItemsResponse['items'] = [];
	let searchError: string | null = null;
	if (searchQuery) {
		try {
			const query = new URLSearchParams({ kind: 'movie', q: searchQuery, limit: '12' });
			const response = await createApiClient(fetch).get<ItemsResponse>(
				`/api/v1/items?${query}`,
				token
			);
			searchResults = response.items;
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 401) {
				redirect(303, '/login');
			}
			searchError = 'No se pudieron buscar películas.';
		}
	}

	return {
		collection,
		isAdmin,
		searchQuery,
		searchResults,
		searchError,
		manage: isAdmin && (url.searchParams.get('gestionar') === '1' || searchQuery !== '')
	};
};

export const actions: Actions = {
	update: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = collectionID(params);
		if (id === null) {
			return fail(404, { manage: true, message: 'Colección no encontrada.' });
		}
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const overview = String(form.get('overview') ?? '').trim();
		try {
			await createApiClient(fetch).updateCollection(id, { name, overview }, token);
		} catch (cause) {
			return manageFailure(cause);
		}
		return { success: true, manage: true, message: 'Colección actualizada.' };
	},

	poster: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = collectionID(params);
		if (id === null) {
			return fail(404, { manage: true, message: 'Colección no encontrada.' });
		}
		const form = await request.formData();
		const rawKind = String(form.get('kind') ?? '').trim();
		const posterURL = form.get('poster_url');
		const backdropURL = form.get('backdrop_url');
		const kind =
			rawKind || (backdropURL !== null && posterURL === null ? 'backdrop' : 'poster');
		const url = String(
			form.get('url') ?? (kind === 'backdrop' ? backdropURL : posterURL) ?? ''
		).trim();
		if (kind !== 'poster' && kind !== 'backdrop') {
			return fail(400, { manage: true, message: 'Tipo de imagen no válido.' });
		}
		try {
			await createApiClient(fetch).updateCollection(
				id,
				kind === 'poster' ? { poster_url: url } : { backdrop_url: url },
				token
			);
		} catch (cause) {
			return manageFailure(cause);
		}
		return {
			success: true,
			manage: true,
			message: kind === 'poster' ? 'Cartel actualizado.' : 'Fondo actualizado.'
		};
	},

	addItem: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = collectionID(params);
		if (id === null) {
			return fail(404, { manage: true, message: 'Colección no encontrada.' });
		}
		const form = await request.formData();
		const mediaItemID = Number(
			String(form.get('media_item_id') ?? form.get('itemId') ?? '')
		);
		if (!Number.isInteger(mediaItemID) || mediaItemID <= 0) {
			return fail(400, { manage: true, message: 'Película no válida.' });
		}
		try {
			await createApiClient(fetch).addCollectionItem(id, mediaItemID, token);
		} catch (cause) {
			return manageFailure(cause);
		}
		return { success: true, manage: true, message: 'Película añadida a la colección.' };
	},

	removeItem: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = collectionID(params);
		if (id === null) {
			return fail(404, { manage: true, message: 'Colección no encontrada.' });
		}
		const form = await request.formData();
		const mediaItemID = Number(
			String(form.get('media_item_id') ?? form.get('itemId') ?? '')
		);
		if (!Number.isInteger(mediaItemID) || mediaItemID <= 0) {
			return fail(400, { manage: true, message: 'Película no válida.' });
		}
		try {
			await createApiClient(fetch).removeCollectionItem(id, mediaItemID, token);
		} catch (cause) {
			return manageFailure(cause);
		}
		return { success: true, manage: true, message: 'Película quitada de la colección.' };
	},

	move: async ({ request, locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = collectionID(params);
		if (id === null) {
			return fail(404, { manage: true, message: 'Colección no encontrada.' });
		}
		const form = await request.formData();
		const mediaItemID = Number(
			String(form.get('media_item_id') ?? form.get('itemId') ?? '')
		);
		const direction = String(form.get('direction') ?? '');
		if (
			!Number.isInteger(mediaItemID) ||
			mediaItemID <= 0 ||
			(direction !== 'up' && direction !== 'down')
		) {
			return fail(400, { manage: true, message: 'Movimiento no válido.' });
		}

		const api = createApiClient(fetch);
		let collection: CollectionDetail;
		try {
			collection = await api.getCollection(id, token);
		} catch (cause) {
			return manageFailure(cause);
		}
		if (!collection.items.some((item) => item.id === mediaItemID)) {
			return fail(400, { manage: true, message: 'La película no pertenece a la colección.' });
		}
		const ordered = moveItem(
			collection.items.map((item) => item.id),
			mediaItemID,
			direction
		);
		try {
			await api.reorderCollection(id, ordered, token);
		} catch (cause) {
			return manageFailure(cause);
		}
		return { success: true, manage: true, message: 'Orden actualizado.' };
	},

	delete: async ({ locals, fetch, params }) => {
		const token = requireSessionToken(locals);
		const id = collectionID(params);
		if (id === null) {
			return fail(404, { manage: true, message: 'Colección no encontrada.' });
		}
		try {
			await createApiClient(fetch).deleteCollection(id, token);
		} catch (cause) {
			return manageFailure(cause);
		}
		redirect(303, '/collections');
	}
};
