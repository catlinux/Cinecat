import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';

const FS_ERROR = 'No se pudo consultar las carpetas del servidor.';

function noStore(body: unknown, status = 200): Response {
	return json(body, {
		status,
		headers: { 'Cache-Control': 'no-store' }
	});
}

export const GET: RequestHandler = async ({ url, locals, fetch }) => {
	const token = locals.sessionToken;
	if (!token) {
		return noStore({ error: 'No autorizado.' }, 401);
	}

	try {
		const path = url.searchParams.get('path');
		return noStore(await createApiClient(fetch).browseFS(path, token));
	} catch (reason) {
		if (reason instanceof ApiError) {
			if (reason.status === 400 || reason.status === 401 || reason.status === 403) {
				return noStore({ error: reason.message }, reason.status);
			}
			if (reason.status === 404) {
				return noStore({ error: reason.message || 'La carpeta no existe.' }, 404);
			}
		}
		return noStore({ error: FS_ERROR }, 502);
	}
};
