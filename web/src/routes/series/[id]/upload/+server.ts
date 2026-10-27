import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiBaseUrl } from '$lib/config';

const UPLOAD_ERROR = 'No se pudo contactar con el servidor para subir el capítulo.';

function noStore(body: unknown, status = 200): Response {
	return json(body, {
		status,
		headers: { 'Cache-Control': 'no-store' }
	});
}

export const PUT: RequestHandler = async ({ request, url, params, locals, fetch }) => {
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
	const base = apiBaseUrl.replace(/\/+$/, '');
	const targetPath = `/api/v1/series/${encodeURIComponent(params.id ?? '')}/files${
		query.toString() ? `?${query}` : ''
	}`;
	const target = base ? `${base}${targetPath}` : new URL(targetPath, url.origin).toString();

	try {
		const response = await fetch(target, {
			method: 'PUT',
			body: request.body,
			duplex: 'half',
			headers: {
				Authorization: `Bearer ${token}`,
				'Content-Length': contentLength,
				'Content-Type': 'application/octet-stream',
				Accept: 'application/json'
			}
		} as RequestInit & { duplex: 'half' });
		return new Response(response.body, {
			status: response.status,
			headers: {
				'Content-Type': response.headers.get('Content-Type') ?? 'application/json; charset=utf-8',
				'Cache-Control': 'no-store'
			}
		});
	} catch {
		return noStore({ error: UPLOAD_ERROR }, 502);
	}
};
