import { json } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, request, locals, fetch }) => {
	const token = locals.sessionToken;
	if (!token) {
		return json({ error: 'falta la sesión' }, { status: 401 });
	}

	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		return json({ error: 'serie no encontrada' }, { status: 404 });
	}

	let body: unknown;
	try {
		body = JSON.parse(await request.text());
	} catch {
		return json({ error: 'petición no válida' }, { status: 400 });
	}
	if (typeof body !== 'object' || body === null || Array.isArray(body)) {
		return json({ error: 'petición no válida' }, { status: 400 });
	}
	const enabled = (body as Record<string, unknown>).enabled;
	if (typeof enabled !== 'boolean') {
		return json({ error: 'enabled no válido' }, { status: 400 });
	}

	try {
		const response = await createApiClient(fetch).setSeriesAutoplay(id, enabled, token);
		return json({ enabled: response.enabled });
	} catch (error) {
		if (error instanceof ApiError) {
			if (error.status === 401) {
				return json({ error: 'sesión no válida' }, { status: 401 });
			}
			if (error.status === 400) {
				return json({ error: 'ajuste no válido' }, { status: 400 });
			}
			if (error.status === 404) {
				return json({ error: 'serie no encontrada' }, { status: 404 });
			}
		}
		return json({ error: 'no se pudo guardar la reproducción automática' }, { status: 502 });
	}
};
