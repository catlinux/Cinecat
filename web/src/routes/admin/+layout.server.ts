import { error, redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';

const ADMIN_MESSAGE = 'Esta sección es solo para administradores.';

export const load: LayoutServerLoad = async ({ locals, fetch, url }) => {
	const token = requireSessionToken(locals);

	let user;
	try {
		user = await createApiClient(fetch).getCurrentUser(token);
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, 'No se pudo comprobar tu cuenta. Inténtalo de nuevo.');
	}

	if (user.role !== 'admin') {
		error(403, ADMIN_MESSAGE);
	}

	return {
		adminUser: user,
		pathname: url?.pathname ?? ''
	};
};
