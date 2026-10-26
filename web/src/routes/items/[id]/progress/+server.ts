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
		return json({ error: 'ítem no encontrado' }, { status: 404 });
	}

	let body: unknown;
	try {
		body = JSON.parse(await request.text());
	} catch {
		return json({ error: 'petición no válida' }, { status: 400 });
	}
	if (typeof body !== 'object' || body === null) {
		return json({ error: 'petición no válida' }, { status: 400 });
	}

	const record = body as Record<string, unknown>;
	const position = record.position_seconds;
	const duration = record.duration_seconds;
	if (
		typeof position !== 'number' ||
		typeof duration !== 'number' ||
		!Number.isFinite(position) ||
		!Number.isFinite(duration) ||
		position < 0 ||
		duration < 0
	) {
		return json({ error: 'position_seconds y duration_seconds no son válidos' }, { status: 400 });
	}

	try {
		await createApiClient(fetch).putProgress(id, position, duration, token);
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) {
			return json({ error: 'sesión no válida' }, { status: 401 });
		}
		if (err instanceof ApiError && err.status === 404) {
			return json({ error: 'ítem no encontrado' }, { status: 404 });
		}
		return json({ error: 'no se pudo guardar el progreso' }, { status: 502 });
	}

	return new Response(null, { status: 204 });
};
