import { json } from '@sveltejs/kit';
import type { MusicPlayResponse } from '$lib/api';
import { apiBaseUrl } from '$lib/config';
import type { RequestHandler } from './$types';

const UNAUTHORIZED = { error: 'No autorizado.' };
const NOT_FOUND = { error: 'Pista no encontrada.' };
const FORBIDDEN = { error: 'No se puede reproducir esta pista.' };
const BAD_GATEWAY = { error: 'No se pudo preparar la reproducción.' };

function errorResponse(body: { error: string }, status: number): Response {
	return json(body, { status, headers: { 'Cache-Control': 'no-store' } });
}

export const GET: RequestHandler = async ({ params, locals, fetch, request }) => {
	const token = locals.sessionToken;
	if (!token) {
		return errorResponse(UNAUTHORIZED, 401);
	}

	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		return errorResponse(NOT_FOUND, 404);
	}

	const base = apiBaseUrl.replace(/\/+$/, '') || new URL(request.url).origin;
	let response: Response;
	try {
		response = await fetch(`${base}/api/v1/music/tracks/${id}/play`, {
			headers: {
				Accept: 'application/json',
				Authorization: `Bearer ${token}`
			},
			cache: 'no-store'
		});
	} catch {
		return errorResponse(BAD_GATEWAY, 502);
	}

	if (response.status === 401) {
		return errorResponse(UNAUTHORIZED, 401);
	}
	if (response.status === 403) {
		return errorResponse(FORBIDDEN, 403);
	}
	if (response.status === 404) {
		return errorResponse(NOT_FOUND, 404);
	}
	if (!response.ok) {
		return errorResponse(BAD_GATEWAY, 502);
	}

	let payload: MusicPlayResponse;
	try {
		payload = (await response.json()) as MusicPlayResponse;
	} catch {
		return errorResponse(BAD_GATEWAY, 502);
	}
	if (
		typeof payload?.url !== 'string' ||
		typeof payload?.mime !== 'string' ||
		typeof payload?.expires_at !== 'string' ||
		typeof payload?.playable !== 'boolean'
	) {
		return errorResponse(BAD_GATEWAY, 502);
	}
	return json(payload, { status: 200, headers: { 'Cache-Control': 'no-store' } });
};
