import { json } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import type { RequestHandler } from './$types';

const UPDATE_ERROR = 'No se pudo consultar el estado de la actualización.';

function noStore(body: unknown, status = 200): Response {
	return json(body, {
		status,
		headers: { 'Cache-Control': 'no-store' }
	});
}

export const GET: RequestHandler = async ({ locals, fetch }) => {
	if (!locals.sessionToken) {
		return noStore({ error: 'No autorizado.' }, 401);
	}

	try {
		return noStore(await createApiClient(fetch).getUpdate(locals.sessionToken));
	} catch (reason) {
		if (
			reason instanceof ApiError &&
			(reason.status === 401 || reason.status === 403 || reason.status === 404)
		) {
			return noStore({ error: reason.message }, reason.status);
		}
		return noStore({ error: UPDATE_ERROR }, 502);
	}
};
