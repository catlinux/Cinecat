import { json } from '@sveltejs/kit';
import { apiBaseUrl } from '$lib/config';
import type { RequestHandler } from './$types';

const HEADERS = [
	'Content-Type',
	'Cache-Control',
	'ETag',
	'Last-Modified',
	'Content-Range',
	'Accept-Ranges',
	'Content-Length'
];

function responseHeaders(source: Response): Headers {
	const headers = new Headers();
	for (const name of HEADERS) {
		const value = source.headers.get(name);
		if (value) {
			headers.set(name, value);
		}
	}
	if (!headers.get('Content-Type')) {
		headers.set('Content-Type', 'application/octet-stream');
	}
	return headers;
}

function errorResponse(message: string, status: number): Response {
	return json({ error: message }, { status, headers: { 'Cache-Control': 'no-store' } });
}

export const GET: RequestHandler = async ({ params, locals, fetch, request }) => {
	const token = locals.sessionToken;
	if (!token) {
		return errorResponse('No autorizado.', 401);
	}

	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		return errorResponse('Foto no encontrada.', 404);
	}

	const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
	for (const name of ['Range', 'If-None-Match', 'If-Modified-Since', 'If-Range']) {
		const value = request.headers.get(name);
		if (value) {
			headers[name] = value;
		}
	}
	const base = apiBaseUrl.replace(/\/+$/, '') || new URL(request.url).origin;
	let response: Response;
	try {
		response = await fetch(`${base}/api/v1/photos/${id}/original`, {
			headers,
			cache: 'no-store'
		});
	} catch {
		return errorResponse('No se pudo cargar la foto.', 502);
	}

	if (response.status === 304) {
		return new Response(null, {
			status: 304,
			headers: responseHeaders(response)
		});
	}
	if (response.status === 401) {
		return errorResponse('No autorizado.', 401);
	}
	if (response.status === 404) {
		return errorResponse('Foto no encontrada.', 404);
	}
	if (!response.ok) {
		return errorResponse('No se pudo cargar la foto.', 502);
	}
	return new Response(response.body, {
		status: response.status,
		headers: responseHeaders(response)
	});
};
