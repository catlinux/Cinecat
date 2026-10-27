import type { UserListKind, UserListName, ListsBlockKind } from './api';

export interface ListsPageFilters {
	kind: ListsBlockKind | null;
	page: number;
}

export function sanitizeRedirectTo(raw: unknown, fallback = '/'): string {
	const value = String(raw ?? '').trim();
	if (
		value === '' ||
		!value.startsWith('/') ||
		value.startsWith('//') ||
		value.includes('\\') ||
		/[\u0000-\u001f\u007f]/.test(value)
	) {
		return fallback;
	}
	try {
		const url = new URL(value, 'http://cinecat.local');
		if (url.origin !== 'http://cinecat.local') {
			return fallback;
		}
		return `${url.pathname}${url.search}${url.hash}`;
	} catch {
		return fallback;
	}
}

export function listToggleLabel(list: UserListName, active: boolean): string {
	if (list === 'favorite') {
		return active ? 'Quitar de favoritos' : 'Añadir a favoritos';
	}
	return active ? 'Quitar de próximas a ver' : 'Añadir a próximas a ver';
}

export function listButtonText(list: UserListName): string {
	return list === 'favorite' ? 'Favorito' : 'Próximas a ver';
}

export function listToggleResult(
	previous: boolean,
	resultType: 'success' | 'failure' | 'error' | 'redirect'
): { active: boolean; error: string | null } {
	if (resultType === 'failure' || resultType === 'error') {
		return { active: previous, error: 'No se pudo actualizar la lista.' };
	}
	return { active: !previous, error: null };
}

export function listEndpointPath(kind: UserListKind, id: number, list: UserListName): string {
	if (kind === 'item') {
		return `/api/v1/items/${id}/lists/${list}`;
	}
	if (kind === 'series') {
		return `/api/v1/series/${id}/lists/${list}`;
	}
	return `/api/v1/collections/${id}/lists/${list}`;
}

export function parseUserList(value: unknown): UserListName | null {
	const text = String(value ?? '').trim();
	return text === 'favorite' || text === 'watchlist' ? text : null;
}

export function parseToggleKind(value: unknown): UserListKind | null {
	const text = String(value ?? '').trim();
	return text === 'item' || text === 'series' || text === 'collection' ? text : null;
}

export function parseListKind(value: unknown): ListsBlockKind | null {
	const text = String(value ?? '').trim();
	return text === 'items' || text === 'series' || text === 'collections' ? text : null;
}

export function readListsPage(url: URL): ListsPageFilters {
	const rawPage = String(url.searchParams.get('page') ?? '').trim();
	const page = /^\d+$/.test(rawPage) ? Number(rawPage) : 1;
	return {
		kind: parseListKind(url.searchParams.get('kind')),
		page: Number.isSafeInteger(page) && page >= 1 ? page : 1
	};
}

export function listPageHref(currentUrl: string, kind: ListsBlockKind, page: number): string {
	const url = new URL(currentUrl, 'http://cinecat.local');
	url.searchParams.set('kind', kind);
	url.searchParams.delete('error');
	if (page > 1) {
		url.searchParams.set('page', String(Math.floor(page)));
	} else {
		url.searchParams.delete('page');
	}
	return `${url.pathname}${url.search}`;
}

export function listSectionCountLabel(count: number, singular: string, plural: string): string {
	const value = Number.isFinite(count) && count > 0 ? Math.floor(count) : 0;
	return `${value} ${value === 1 ? singular : plural}`;
}

export function listPageTitle(list: UserListName): string {
	return list === 'favorite' ? 'Favoritos' : 'Próximas a ver';
}

export function listPageKicker(list: UserListName): string {
	return list === 'favorite' ? 'Tu selección' : 'Para más tarde';
}
