import { error, redirect } from '@sveltejs/kit';
import { ApiError, type VideoBrowseResponse } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { clampPage, pageCount, pageHref } from '$lib/pagination';
import { requireSessionToken } from '$lib/session';
import { DEFAULT_PUBLIC_UI, listPageSize } from '$lib/settings';
import { parseVideoLibraryID, parseVideoPath, parseVideoRoot } from '$lib/videos';
import type { PageServerLoad } from './$types';

const LOAD_ERROR = 'No se pudo cargar la biblioteca de vídeos.';
const NOT_FOUND = 'No se encontró esa carpeta';
const MAX_PAGE = Number.MAX_SAFE_INTEGER;

export const load: PageServerLoad = async ({ params, locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const libraryID = parseVideoLibraryID(params.id);
	if (libraryID === null) {
		error(404, NOT_FOUND);
	}

	const layout =
		typeof parent === 'function' ? await parent() : { ui: DEFAULT_PUBLIC_UI, isAdmin: false };
	const pageSize = listPageSize(layout.ui);
	const rootParam = parseVideoRoot(url.searchParams.get('root'));
	if (!rootParam.ok) {
		error(404, NOT_FOUND);
	}
	const pathParam = parseVideoPath(url.searchParams.get('path'), rootParam.value !== null);
	if (!pathParam.ok) {
		error(404, NOT_FOUND);
	}
	const requestedPage = clampPage(url.searchParams.get('page'), MAX_PAGE);
	const currentUrl = `${url.pathname}${url.search}`;
	const api = createApiClient(fetch);

	let libraryName = 'Vídeos';
	try {
		const libraries = await api.listLibraries(token);
		const library = (libraries.libraries ?? []).find((item) => item.id === libraryID);
		if (library) {
			if (library.kind !== 'clips') {
				error(404, NOT_FOUND);
			}
			libraryName = library.name;
		}
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		if (!(reason instanceof ApiError)) {
			throw reason;
		}
	}

	try {
		if (rootParam.value === null) {
			const response = await api.browseVideoLibrary(libraryID, {}, token);
			if (!('roots' in response)) {
				error(502, LOAD_ERROR);
			}
			return {
				mode: 'roots' as const,
				libraryID,
				libraryName,
				roots: response.roots ?? [],
				page: 1,
				last: 1,
				pageSize,
				currentUrl,
				root: null,
				path: '',
				crumbs: [],
				folders: [],
				items: [],
				total: 0
			};
		}

		const offset = (requestedPage - 1) * pageSize;
		const response = (await api.browseVideoLibrary(
			libraryID,
			{
				root: rootParam.value,
				path: pathParam.value,
				limit: pageSize,
				offset
			},
			token
		)) as VideoBrowseResponse;
		if ('roots' in response) {
			error(502, LOAD_ERROR);
		}
		const last = pageCount(response.total, pageSize);
		const page = clampPage(requestedPage, last);
		if (page !== requestedPage) {
			redirect(303, pageHref(url, page));
		}

		return {
			mode: 'browse' as const,
			libraryID,
			libraryName: response.library?.name || libraryName,
			root: response.root,
			path: response.path,
			crumbs: response.crumbs ?? [],
			folders: response.folders ?? [],
			items: response.items ?? [],
			total: response.total,
			page,
			last,
			pageSize,
			currentUrl
		};
	} catch (reason) {
		if (reason instanceof ApiError) {
			if (reason.status === 401) {
				redirect(303, '/login');
			}
			if (reason.status === 404) {
				error(404, NOT_FOUND);
			}
			error(502, LOAD_ERROR);
		}
		throw reason;
	}
};
