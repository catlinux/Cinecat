import { json } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import {
	listEndpointPath,
	parseToggleKind,
	parseUserList,
	sanitizeRedirectTo
} from '$lib/lists';
import type { RequestHandler } from './$types';

interface ListForm {
	kind: unknown;
	id: unknown;
	list: unknown;
	on: unknown;
	redirectTo: unknown;
}

function parseOn(value: unknown): boolean | null {
	if (value === true || value === '1' || value === 'true' || value === 'on') {
		return true;
	}
	if (value === false || value === '0' || value === 'false') {
		return false;
	}
	return null;
}

function parseObjectID(value: unknown): number | null {
	const number = Number(value);
	return Number.isInteger(number) && number > 0 ? number : null;
}

function isEnhanced(request: Request): boolean {
	return request.headers.get('x-sveltekit-action') === 'true';
}

function wantsJSON(request: Request): boolean {
	const accept = request.headers.get('accept') ?? '';
	const contentType = request.headers.get('content-type') ?? '';
	return accept.includes('application/json') || contentType.includes('application/json');
}

function actionResult(type: 'success' | 'failure' | 'redirect', status: number, data?: unknown, location?: string): Response {
	const payload: Record<string, unknown> = { type, status };
	if (data !== undefined) {
		payload.data = data;
	}
	if (location !== undefined) {
		payload.location = location;
	}
	return new Response(JSON.stringify(payload), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
}

function seeOther(location: string): Response {
	return new Response(null, { status: 303, headers: { Location: location } });
}

function withoutError(redirectTo: string): string {
	const url = new URL(redirectTo, 'http://cinecat.local');
	url.searchParams.delete('error');
	return `${url.pathname}${url.search}${url.hash}`;
}

function withError(redirectTo: string): string {
	const url = new URL(redirectTo, 'http://cinecat.local');
	url.searchParams.set('error', 'lista');
	return `${url.pathname}${url.search}`;
}

export const POST: RequestHandler = async ({ request, locals, fetch }) => {
	const enhanced = isEnhanced(request);
	const jsonMode = !enhanced && wantsJSON(request);
	let form: ListForm;
	try {
		if ((request.headers.get('content-type') ?? '').includes('application/json')) {
			form = (await request.json()) as ListForm;
		} else {
			const data = await request.formData();
			form = {
				kind: data.get('kind'),
				id: data.get('id'),
				list: data.get('list'),
				on: data.get('on'),
				redirectTo: data.get('redirectTo')
			};
		}
	} catch {
		form = { kind: null, id: null, list: null, on: null, redirectTo: null };
	}

	const rawRedirect = String(form.redirectTo ?? '').trim();
	let redirectTo = sanitizeRedirectTo(rawRedirect);
	if (rawRedirect === '') {
		const referer = request.headers.get('referer');
		if (referer) {
			try {
				const url = new URL(referer);
				redirectTo = sanitizeRedirectTo(`${url.pathname}${url.search}`);
			} catch {
				redirectTo = '/';
			}
		}
	}

	const kind = parseToggleKind(form.kind);
	const id = parseObjectID(form.id);
	const list = parseUserList(form.list);
	const on = parseOn(form.on);

	const failure = (status: number, message: string, login = false): Response => {
		if (login) {
			if (enhanced) {
				return actionResult('redirect', 303, undefined, '/login');
			}
			if (jsonMode) {
				return json({ error: message }, { status: 401, headers: { 'Cache-Control': 'no-store' } });
			}
			return seeOther('/login');
		}
		if (enhanced) {
			return actionResult('failure', status, { message });
		}
		if (jsonMode) {
			return json({ error: message }, { status, headers: { 'Cache-Control': 'no-store' } });
		}
		return seeOther(withError(redirectTo));
	};

	if (!locals.sessionToken) {
		return failure(401, 'No autorizado.', true);
	}
	if (!kind || !id || !list || on === null) {
		return failure(400, 'Los datos de la lista no son válidos.');
	}

	try {
		const api = createApiClient(fetch);
		const path = listEndpointPath(kind, id, list);
		if (on) {
			await api.put<void>(path, undefined, locals.sessionToken);
		} else {
			await api.delete(path, locals.sessionToken);
		}
	} catch (error) {
		if (error instanceof ApiError) {
			if (error.status === 401) {
				return failure(401, 'No autorizado.', true);
			}
			if (error.status === 404) {
				return failure(404, 'El elemento ya no existe.');
			}
		}
		return failure(502, 'No se pudo actualizar la lista.');
	}

	if (enhanced) {
		return actionResult('success', 204);
	}
	if (jsonMode) {
		return new Response(null, { status: 204, headers: { 'Cache-Control': 'no-store' } });
	}
	return seeOther(withoutError(redirectTo));
};
