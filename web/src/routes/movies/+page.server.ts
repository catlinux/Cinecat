import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type { GenreCount, GenresResponse, ItemsResponse } from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import {
	clearFiltersHref,
	filterChips,
	movieApiQuery,
	readMovieFilters
} from '$lib/movies';
import { clampPage, pageCount, pageHref } from '$lib/pagination';
import { requireSessionToken } from '$lib/session';
import { DEFAULT_PUBLIC_UI, listPageSize } from '$lib/settings';
import type { PageServerLoad } from './$types';

const ERROR_MESSAGE = 'No se pudo cargar la cartelera. Inténtalo de nuevo en unos momentos.';

function emptyPage(
	filters: ReturnType<typeof readMovieFilters>,
	currentUrl: string,
	isAdmin: boolean,
	pageSize: number,
	message: string
) {
	return {
		items: [],
		total: 0,
		page: 1,
		last: 1,
		pageSize,
		filters,
		genres: [] as GenreCount[],
		isAdmin,
		unmatchedCount: 0,
		filterChips: filterChips(filters, currentUrl),
		clearHref: clearFiltersHref(currentUrl),
		currentUrl,
		errorMessage: message
	};
}

export const load: PageServerLoad = async ({ locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const layout =
		typeof parent === 'function' ? await parent() : { isAdmin: false, ui: DEFAULT_PUBLIC_UI };
	const isAdmin = layout.isAdmin;
	const pageSize = listPageSize(layout.ui);
	const filters = readMovieFilters(url, isAdmin);
	const currentUrl = `${url.pathname}${url.search}`;
	const requestedPage = clampPage(url.searchParams.get('page'), Number.MAX_SAFE_INTEGER);

	let items: ItemsResponse;
	try {
		items = await createApiClient(fetch).get<ItemsResponse>(
			`/api/v1/items?${movieApiQuery(filters, requestedPage, pageSize)}`,
			token
		);
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
		return emptyPage(
			filters,
			currentUrl,
			isAdmin,
			pageSize,
			error instanceof ConfigurationError ? error.message : ERROR_MESSAGE
		);
	}

	const last = pageCount(items.total, pageSize);
	const page = clampPage(requestedPage, last);
	if (page !== requestedPage) {
		redirect(303, pageHref(url, page));
	}

	let genres: GenreCount[] = [];
	try {
		const response = await createApiClient(fetch).get<GenresResponse>(
			'/api/v1/genres?kind=movie',
			token
		);
		genres = response.genres ?? [];
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
	}

	let unmatchedCount = 0;
	if (isAdmin) {
		try {
			const response = await createApiClient(fetch).get<ItemsResponse>(
				'/api/v1/items?kind=movie&limit=1&unmatched=true',
				token
			);
			unmatchedCount = response.total;
		} catch (error) {
			if (error instanceof ApiError && error.status === 401) {
				redirect(303, '/login');
			}
		}
	}

	return {
		items: items.items,
		total: items.total,
		page,
		last,
		pageSize,
		filters,
		genres,
		isAdmin,
		unmatchedCount,
		filterChips: filterChips(filters, currentUrl),
		clearHref: clearFiltersHref(currentUrl),
		currentUrl,
		errorMessage: null
	};
};
