import { fail, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import type { Actions, PageServerLoad } from './$types';

const CODE_FORMAT = 'Ese código no tiene el formato correcto.';
const CODE_INVALID = 'El código no existe o ha caducado. Pide uno nuevo en el dispositivo.';
const TOO_MANY = 'Demasiados intentos. Espera unos minutos.';
const LINK_ERROR = 'No se pudo vincular. Inténtalo de nuevo.';

function nextPathWithCode(code: string): string {
	return code ? `/vincular?codigo=${encodeURIComponent(code)}` : '/vincular';
}

function loginWithNext(path: string): never {
	redirect(303, `/login?next=${encodeURIComponent(path)}`);
}

function actionFailure(reason: unknown, code: string) {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			loginWithNext(nextPathWithCode(code));
		}
		if (reason.status === 400) {
			return fail(400, { message: CODE_FORMAT, code });
		}
		if (reason.status === 404) {
			return fail(404, { message: CODE_INVALID, code });
		}
		if (reason.status === 429) {
			return fail(429, { message: TOO_MANY, code });
		}
	}
	return fail(502, { message: LINK_ERROR, code });
}

async function codeFrom(request: Request): Promise<string> {
	const form = await request.formData();
	return String(form.get('code') ?? '');
}

export const load: PageServerLoad = async ({ locals, url }) => {
	if (!locals.sessionToken) {
		loginWithNext(`${url.pathname}${url.search}`);
	}
	return { initialCode: url.searchParams.get('codigo') ?? '' };
};

export const actions: Actions = {
	preview: async ({ request, locals, fetch, url }) => {
		const code = (await codeFrom(request)) || url.searchParams.get('codigo') || '';
		const token = locals.sessionToken;
		if (!token) {
			loginWithNext(nextPathWithCode(code));
		}

		try {
			const preview = await createApiClient(fetch).approvePairing(code, false, token);
			return {
				success: true,
				step: 'preview' as const,
				code,
				deviceName: preview.device_name,
				client: preview.client,
				expiresAt: preview.expires_at ?? ''
			};
		} catch (reason) {
			return actionFailure(reason, code);
		}
	},

	approve: async ({ request, locals, fetch, url }) => {
		const code = (await codeFrom(request)) || url.searchParams.get('codigo') || '';
		const token = locals.sessionToken;
		if (!token) {
			loginWithNext(nextPathWithCode(code));
		}

		try {
			const approved = await createApiClient(fetch).approvePairing(code, true, token);
			return {
				success: true,
				step: 'approved' as const,
				code,
				deviceName: approved.device_name,
				client: approved.client
			};
		} catch (reason) {
			return actionFailure(reason, code);
		}
	}
};
