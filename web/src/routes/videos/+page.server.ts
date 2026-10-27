import { redirect } from '@sveltejs/kit';
import { ApiError, type Library } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

const LOAD_ERROR = 'No se pudieron cargar las bibliotecas de vídeos.';

export const load: PageServerLoad = async ({ locals, fetch, parent }) => {
	const token = requireSessionToken(locals);
	const layout = typeof parent === 'function' ? await parent() : { isAdmin: false };

	let libraries: Library[] = [];
	let errorMessage: string | null = null;
	try {
		const response = await createApiClient(fetch).listLibraries(token);
		libraries = (response.libraries ?? []).filter(
			(library) => library.kind === 'clips' && (library.item_count ?? 0) > 0
		);
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		errorMessage = LOAD_ERROR;
	}

	if (!errorMessage && libraries.length === 1) {
		redirect(303, `/videos/${libraries[0].id}`);
	}

	return {
		libraries,
		isAdmin: Boolean(layout?.isAdmin),
		errorMessage
	};
};
