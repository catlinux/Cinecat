import type { LayoutServerLoad } from './$types';
import { createApiClient } from '$lib/api-factory';

// Expone en el layout si hay sesión y si el usuario es administrador. La
// consulta del rol solo se hace con sesión; si la API falla, se oculta la
// administración sin romper la página.
export const load: LayoutServerLoad = async ({ locals, fetch }) => {
	if (!locals.sessionToken) {
		return { hasSession: false, isAdmin: false };
	}

	let isAdmin = false;
	try {
		const user = await createApiClient(fetch).getCurrentUser(locals.sessionToken);
		isAdmin = user.role === 'admin';
	} catch {
		isAdmin = false;
	}

	return { hasSession: true, isAdmin };
};
