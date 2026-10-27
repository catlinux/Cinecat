import { redirect } from '@sveltejs/kit';
import { ApiError } from '$lib/api';
import type {
	GenreCount,
	GenresResponse,
	SeriesResponse,
	UploadTargetsResponse
} from '$lib/api';
import { ConfigurationError, createApiClient } from '$lib/api-factory';
import {
	clearSeriesFiltersHref,
	readSeriesFilters,
	seriesApiQuery,
	seriesFilterChips
} from '$lib/series';
import { clampPage, pageCount, pageHref } from '$lib/pagination';
import { requireSessionToken } from '$lib/session';
import { DEFAULT_PUBLIC_UI, listPageSize } from '$lib/settings';
import type { PageServerLoad } from './$types';

const ERROR_MESSAGE = 'No se pudieron cargar las series. Inténtalo de nuevo en unos momentos.';

function emptyPage(
	filters: ReturnType<typeof readSeriesFilters>,
	currentUrl: string,
	isAdmin: boolean,
	pageSize: number,
	message: string
) {
	return {
		series: [],
		total: 0,
		page: 1,
		last: 1,
		pageSize,
		filters,
		genres: [] as GenreCount[],
		isAdmin,
		unmatchedCount: 0,
		filterChips: seriesFilterChips(filters, currentUrl),
		clearHref: clearSeriesFiltersHref(currentUrl),
		currentUrl,
		errorMessage: message,
		uploadTargets: null as UploadTargetsResponse | null
	};
}

export const load: PageServerLoad = async ({ locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const layout =
		typeof parent === 'function' ? await parent() : { isAdmin: false, ui: DEFAULT_PUBLIC_UI };
	const isAdmin = Boolean(layout?.isAdmin);
	const pageSize = listPageSize(layout.ui);
	const filters = readSeriesFilters(url, isAdmin);
	const currentUrl = `${url.pathname}${url.search}`;
	const requestedPage = clampPage(url.searchParams.get('page'), Number.MAX_SAFE_INTEGER);
	const api = createApiClient(fetch);

	let response: SeriesResponse;
	try {
		response = await api.get<SeriesResponse>(
			`/api/v1/series?${seriesApiQuery(filters, requestedPage, pageSize)}`,
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

	const last = pageCount(response.total, pageSize);
	const page = clampPage(requestedPage, last);
	if (page !== requestedPage) {
		redirect(303, pageHref(url, page));
	}

	let genres: GenreCount[] = [];
	try {
		const response = await api.getSeriesGenres(token);
		genres = response.genres ?? [];
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
	}

	let unmatchedCount = 0;
	if (isAdmin) {
		try {
			const unmatched = await api.get<SeriesResponse>(
				'/api/v1/series?limit=1&unmatched=true',
				token
			);
			unmatchedCount = unmatched.total;
		} catch (error) {
			if (error instanceof ApiError && error.status === 401) {
				redirect(303, '/login');
			}
		}
	}

	let uploadTargets: UploadTargetsResponse | null = null;
	if (isAdmin) {
		try {
			uploadTargets = await api.getUploadTargets(token);
		} catch (error) {
			if (error instanceof ApiError && error.status === 401) {
				redirect(303, '/login');
			}
		}
	}

	return {
		series: response.series,
		total: response.total,
		page,
		last,
		pageSize,
		filters,
		genres,
		isAdmin,
		unmatchedCount,
		filterChips: seriesFilterChips(filters, currentUrl),
		clearHref: clearSeriesFiltersHref(currentUrl),
		currentUrl,
		errorMessage: null,
		uploadTargets
	};
};
