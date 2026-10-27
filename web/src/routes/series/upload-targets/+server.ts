import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiBaseUrl } from '$lib/config';

const TARGETS_ERROR = 'No se pudieron consultar los destinos de subida.';

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

	const base = apiBaseUrl.replace(/\/+$/, '');
	const targetPath = '/api/v1/uploads/targets';
	const target = base ? `${base}${targetPath}` : new URL(targetPath, url.origin).toString();

	try {
		const response = await fetch(target, {
			headers: {
				Authorization: `Bearer ${token}`,
				Accept: 'application/json'
			}
		});
		return new Response(response.body, {
			status: response.status,
			headers: {
				'Content-Type': response.headers.get('Content-Type') ?? 'application/json; charset=utf-8',
				'Cache-Control': 'no-store'
			}
		});
	} catch {
		return noStore({ error: TARGETS_ERROR }, 502);
	}
};
