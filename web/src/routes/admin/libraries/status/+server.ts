import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';

const STATUS_ERROR = 'No se pudo consultar el estado del escaneo.';

export const GET: RequestHandler = async ({ locals, fetch }) => {
	if (!locals.sessionToken) {
		return json({ message: 'No autorizado.' }, { status: 401 });
	}
	const token = requireSessionToken(locals);

	try {
		return json(await createApiClient(fetch).getScanStatus(token));
	} catch (reason) {
		if (reason instanceof ApiError && (reason.status === 401 || reason.status === 403)) {
			return json({ message: reason.message }, { status: reason.status });
		}
		return json({ message: STATUS_ERROR }, { status: 502 });
	}
};
