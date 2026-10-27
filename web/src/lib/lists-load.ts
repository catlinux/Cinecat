import { redirect } from '@sveltejs/kit';
import type { Collection, MediaItem, Series, UserListName } from './api';
import { ApiError } from './api';
import { createApiClient } from './api-factory';
import { readListsPage, listPageHref } from './lists';
import { pageCount } from './pagination';
import { requireSessionToken } from './session';
import { DEFAULT_PUBLIC_UI, listPageSize } from './settings';

const LOAD_ERROR = 'No se pudieron cargar tus listas. Inténtalo de nuevo en unos momentos.';

export async function loadUserListPage(
	list: UserListName,
	event: {
		locals: App.Locals;
		fetch: typeof fetch;
		url: URL;
		parent?: () => Promise<unknown>;
	}
) {
	const token = requireSessionToken(event.locals);
	const layout = (
		typeof event.parent === 'function'
			? await event.parent()
			: { ui: DEFAULT_PUBLIC_UI }
	) as { ui?: typeof DEFAULT_PUBLIC_UI };
	const pageSize = listPageSize(layout.ui);
	const filters = readListsPage(event.url);
	const page = filters.kind ? filters.page : 1;
	const offset = filters.kind ? (page - 1) * pageSize : 0;
	const currentUrl = `${event.url.pathname}${event.url.search}`;
	const emptyTotals = { items: 0, series: 0, collections: 0 };
	let items: MediaItem[] = [];
	let series: Series[] = [];
	let collections: Collection[] = [];
	let totals = emptyTotals;
	let selectedTotal = 0;
	let errorMessage: string | null = null;

	try {
		const response = await createApiClient(event.fetch).getLists(
			list,
			{ kind: filters.kind ?? undefined, limit: pageSize, offset },
			token
		);
		items = response.items ?? [];
		series = response.series ?? [];
		collections = response.collections ?? [];
		totals = response.totals ?? {
			items: items.length,
			series: series.length,
			collections: collections.length
		};
		selectedTotal =
			filters.kind === 'items'
				? totals.items
				: filters.kind === 'series'
					? totals.series
					: filters.kind === 'collections'
						? totals.collections
						: 0;
	} catch (error) {
		if (error instanceof ApiError && error.status === 401) {
			redirect(303, '/login');
		}
		errorMessage = LOAD_ERROR;
	}

	const last = filters.kind ? pageCount(selectedTotal, pageSize) : 1;
	if (!errorMessage && filters.kind && page > last) {
		redirect(303, listPageHref(currentUrl, filters.kind, last));
	}

	return {
		list,
		kind: filters.kind,
		page,
		last,
		total: selectedTotal,
		pageSize,
		items,
		series,
		collections,
		totals,
		currentUrl,
		errorMessage,
		actionError: event.url.searchParams.get('error') === 'lista'
	};
}
