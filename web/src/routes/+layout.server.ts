import type { LayoutServerLoad } from './$types';
import { createApiClient } from '$lib/api-factory';
import { DEFAULT_PUBLIC_UI, parsePublicUI, type PublicUI } from '$lib/settings';
import { webVersion } from '$lib/version';

// Expone en el layout si hay sesión y si el usuario es administrador. La
// consulta del rol solo se hace con sesión; si la API falla, se oculta la
// administración sin romper la página. La versión del servidor es pública y
// también se consulta sin sesión; si no responde, el pie usa la de la web.
export const load: LayoutServerLoad = async ({ locals, fetch, url }) => {
	const pathname = url?.pathname ?? '';
	const isHome = pathname === '/';
	const activeCategory:
		| 'movies'
		| 'collections'
		| 'series'
		| 'videos'
		| 'music'
		| 'favorites'
		| 'watchlist'
		| null =
		pathname === '/favorites'
			? 'favorites'
			: pathname === '/watchlist'
				? 'watchlist'
				: pathname === '/music' || pathname.startsWith('/music/')
					? 'music'
					: pathname === '/videos' || pathname.startsWith('/videos/')
						? 'videos'
						: pathname === '/movies' ||
							  pathname.startsWith('/movies/') ||
							  pathname.startsWith('/items/')
							? 'movies'
							: pathname === '/collections' || pathname.startsWith('/collections/')
								? 'collections'
								: pathname === '/series' || pathname.startsWith('/series/')
									? 'series'
									: null;
	let serverVersion: string | null = null;
	let serverCommit: string | null = null;
	try {
		const info = await createApiClient(fetch).getServerVersion();
		serverVersion = info.server;
		serverCommit = info.commit || null;
	} catch {
		serverVersion = null;
	}

	let ui: PublicUI = DEFAULT_PUBLIC_UI;
	if (!locals.sessionToken) {
		locals.publicUI = ui;
		return {
			hasSession: false,
			isAdmin: false,
			hasClips: false,
			hasMusic: false,
			isHome,
			activeCategory,
			serverVersion,
			serverCommit,
			webVersion,
			ui
		};
	}

	try {
		ui = parsePublicUI(await createApiClient(fetch).getPublicSettings(locals.sessionToken));
	} catch {
		ui = DEFAULT_PUBLIC_UI;
	}
	locals.publicUI = ui;

	let isAdmin = false;
	try {
		const user = await createApiClient(fetch).getCurrentUser(locals.sessionToken);
		isAdmin = user.role === 'admin';
	} catch {
		isAdmin = false;
	}

	let hasClips = false;
	let hasMusic = false;
	try {
		const libraries = await createApiClient(fetch).listLibraries(locals.sessionToken);
		hasClips = (libraries.libraries ?? []).some(
			(library) => library.kind === 'clips' && (library.item_count ?? 0) > 0
		);
		hasMusic = (libraries.libraries ?? []).some(
			(library) => library.kind === 'music' && (library.item_count ?? 0) > 0
		);
	} catch {
		hasClips = false;
		hasMusic = false;
	}

	return {
		hasSession: true,
		isAdmin,
		hasClips,
		hasMusic,
		isHome,
		activeCategory,
		serverVersion,
		serverCommit,
		webVersion,
		ui
	};
};
