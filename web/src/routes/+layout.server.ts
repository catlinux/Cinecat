import type { LayoutServerLoad } from './$types';
import { createApiClient } from '$lib/api-factory';
import { webVersion } from '$lib/version';

// Expone en el layout si hay sesión y si el usuario es administrador. La
// consulta del rol solo se hace con sesión; si la API falla, se oculta la
// administración sin romper la página. La versión del servidor es pública y
// también se consulta sin sesión; si no responde, el pie usa la de la web.
export const load: LayoutServerLoad = async ({ locals, fetch }) => {
	let serverVersion: string | null = null;
	let serverCommit: string | null = null;
	try {
		const info = await createApiClient(fetch).getServerVersion();
		serverVersion = info.server;
		serverCommit = info.commit || null;
	} catch {
		serverVersion = null;
	}

	if (!locals.sessionToken) {
		return { hasSession: false, isAdmin: false, serverVersion, serverCommit, webVersion };
	}

	let isAdmin = false;
	try {
		const user = await createApiClient(fetch).getCurrentUser(locals.sessionToken);
		isAdmin = user.role === 'admin';
	} catch {
		isAdmin = false;
	}

	return { hasSession: true, isAdmin, serverVersion, serverCommit, webVersion };
};
