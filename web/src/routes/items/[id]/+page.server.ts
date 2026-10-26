import { error, fail, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { Candidate } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
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
		const item = await createApiClient(fetch).getItem(id, token);
		return {
			item,
			isAdmin,
			corregir: isAdmin && url.searchParams.get('corregir') === '1'
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

export const actions: Actions = {
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
	}
};
