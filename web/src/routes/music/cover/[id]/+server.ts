import { json } from '@sveltejs/kit';
import { apiBaseUrl } from '$lib/config';
import type { RequestHandler } from './$types';

const TRANSPARENT_PNG = Uint8Array.from(
	atob('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII='),
	(character) => character.charCodeAt(0)
);

function transparentPNG(): Response {
	return new Response(TRANSPARENT_PNG, {
		status: 200,
		headers: {
			'Content-Type': 'image/png',
			'Cache-Control': 'private, max-age=60'
		}
	});
}

function imageHeaders(source: Response): Headers {
	const headers = new Headers();
	headers.set('Content-Type', source.headers.get('Content-Type') || 'image/jpeg');
	headers.set('Cache-Control', 'private, max-age=86400');
	for (const name of ['ETag', 'Last-Modified']) {
		const value = source.headers.get(name);
		if (value) {
			headers.set(name, value);
		}
	}
	return headers;
}

export const GET: RequestHandler = async ({ params, locals, fetch, request }) => {
	const token = locals.sessionToken;
	if (!token) {
		return json(
			{ error: 'No autorizado.' },
			{ status: 401, headers: { 'Cache-Control': 'no-store' } }
		);
	}

	const id = Number(params.id);
	if (!Number.isInteger(id) || id <= 0) {
		return transparentPNG();
	}

	const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
	const ifNoneMatch = request.headers.get('if-none-match');
	if (ifNoneMatch) {
		headers['If-None-Match'] = ifNoneMatch;
	}
	const base = apiBaseUrl.replace(/\/+$/, '') || new URL(request.url).origin;
	let response: Response;
	try {
		response = await fetch(`${base}/api/v1/music/albums/${id}/cover`, {
			headers,
			cache: 'no-store'
		});
	} catch {
		return transparentPNG();
	}

	if (response.status === 304) {
		return new Response(null, { status: 304, headers: imageHeaders(response) });
	}
	if (!response.ok) {
		return transparentPNG();
	}
	const body = await response.arrayBuffer();
	if (body.byteLength === 0) {
		return transparentPNG();
	}
	return new Response(body, { status: 200, headers: imageHeaders(response) });
};
