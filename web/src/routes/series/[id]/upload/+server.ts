import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiBaseUrl } from '$lib/config';
import { proxyUpload } from '$lib/server/upload-proxy';

const UPLOAD_ERROR = 'No se pudo contactar con el servidor para subir el capítulo.';

function noStore(body: unknown, status = 200): Response {
	return json(body, {
		status,
		headers: { 'Cache-Control': 'no-store' }
	});
}

export const PUT: RequestHandler = async ({ request, url, params, locals }) => {
	const token = locals.sessionToken;
	if (!token) {
		return noStore({ error: 'No autorizado.' }, 401);
	}
	const contentLength = request.headers.get('content-length')?.trim() ?? '';
	if (!/^\d+$/.test(contentLength)) {
		return noStore({ error: 'Falta la cabecera Content-Length.' }, 411);
	}

	const query = new URLSearchParams();
	for (const key of ['season', 'episode', 'name']) {
		const value = url.searchParams.get(key);
		if (value !== null) {
			query.set(key, value);
		}
	}
	const targetPath = `/api/v1/series/${encodeURIComponent(params.id ?? '')}/files${
		query.toString() ? `?${query}` : ''
	}`;
	try {
		return await proxyUpload(request, targetPath, {
			token,
			contentLength,
			errorMessage: UPLOAD_ERROR,
			origin: url.origin,
			publicBase: apiBaseUrl
		});
	} catch {
		return noStore({ error: UPLOAD_ERROR }, 502);
	}
};
