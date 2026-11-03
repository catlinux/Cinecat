import { error, redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { clampPage, pageCount, pageHref } from '$lib/pagination';
import { parsePhotoFolder, parsePhotoSort, photoBreadcrumbs } from '$lib/photos';
import { requireSessionToken } from '$lib/session';
import { DEFAULT_PUBLIC_UI, listPageSize } from '$lib/settings';
import type { PageServerLoad } from './$types';

const LOAD_ERROR = 'No se pudo cargar la biblioteca de fotos.';
const NOT_FOUND = 'No se encontró esa carpeta.';
const MAX_PAGE = Number.MAX_SAFE_INTEGER;

export const load: PageServerLoad = async ({ params, locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const libraryID = Number(params.libraryId);
	if (!Number.isInteger(libraryID) || libraryID <= 0) {
		error(404, NOT_FOUND);
	}

	const layout =
		typeof parent === 'function' ? await parent() : { ui: DEFAULT_PUBLIC_UI, isAdmin: false };
	const pageSize = listPageSize(layout.ui);
	const folder = parsePhotoFolder(url.searchParams.get('folder'));
	if (folder === null) {
		error(404, NOT_FOUND);
	}
	const sort = parsePhotoSort(url.searchParams.get('sort'));
	const requestedPage = clampPage(url.searchParams.get('page'), MAX_PAGE);
	const currentUrl = `${url.pathname}${url.search}`;
	const api = createApiClient(fetch);

	let libraryName = 'Fotos';
	try {
		const libraries = await api.listLibraries(token);
		const library = (libraries.libraries ?? []).find((item) => item.id === libraryID);
		if (!library || library.kind !== 'photos') {
			error(404, NOT_FOUND);
		}
		libraryName = library.name;
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason instanceof ApiError) {
			error(502, LOAD_ERROR);
		}
		throw reason;
	}

	try {
		const response = await api.getPhotos(
			{
				library_id: libraryID,
				folder,
				sort,
				limit: pageSize,
				offset: (requestedPage - 1) * pageSize
			},
			token
		);
		const last = pageCount(response.total, pageSize);
		const page = clampPage(requestedPage, last);
		if (page !== requestedPage) {
			redirect(303, pageHref(url, page));
		}
		return {
			libraryID,
			libraryName,
			folder,
			crumbs: photoBreadcrumbs(folder),
			items: response.items ?? [],
			folders: response.folders ?? [],
			total: response.total,
			sort,
			page,
			last,
			pageSize,
			currentUrl,
			errorMessage: null
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
