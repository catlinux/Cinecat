import { json } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { ItemPreferences } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import type { RequestHandler } from './$types';

const QUALITY_VALUES = new Set(['auto', 'original', '480', '720', '1080']);
const SUBTITLE_PATTERN = /^([ex]\d+)?$/;

function invalid(message: string) {
	return json({ error: message }, { status: 400 });
}

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
		return invalid('petición no válida');
	}
	if (typeof body !== 'object' || body === null || Array.isArray(body)) {
		return invalid('petición no válida');
	}

	const record = body as Record<string, unknown>;
	const changes: Partial<ItemPreferences> = {};
	for (const [key, value] of Object.entries(record)) {
		switch (key) {
			case 'audio_index':
				if (value !== null && (typeof value !== 'number' || !Number.isInteger(value) || value < 0)) {
					return invalid('audio_index no válido');
				}
				changes.audio_index = value as number | null;
				break;
			case 'subtitle_id':
				if (value !== null && (typeof value !== 'string' || !SUBTITLE_PATTERN.test(value))) {
					return invalid('subtitle_id no válido');
				}
				changes.subtitle_id = value as string | null;
				break;
			case 'quality':
				if (
					value !== null &&
					(typeof value !== 'string' || !QUALITY_VALUES.has(value))
				) {
					return invalid('quality no válida');
				}
				changes.quality = value as string | null;
				break;
			default:
				return invalid('campo desconocido');
		}
	}

	if (Object.keys(changes).length > 0) {
		try {
			await createApiClient(fetch).putPreferences(id, changes, token);
		} catch (err) {
			if (err instanceof ApiError && (err.status === 401 || err.status === 404)) {
				return json({ error: 'no se pudieron guardar las preferencias' }, { status: err.status });
			}
			return json({ error: 'no se pudieron guardar las preferencias' }, { status: 502 });
		}
	}

	return new Response(null, { status: 204 });
};
