import { redirect } from '@sveltejs/kit';
import type { MusicAlbum, MusicArtist, MusicTrack } from '$lib/api';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { readMusicFilters } from '$lib/music';
import { clampPage, pageCount, pageHref } from '$lib/pagination';
import { requireSessionToken } from '$lib/session';
import { DEFAULT_PUBLIC_UI, listPageSize } from '$lib/settings';
import type { PageServerLoad } from './$types';

const LOAD_ERROR = 'No se pudo cargar la música. Inténtalo de nuevo en unos momentos.';

export const load: PageServerLoad = async ({ locals, fetch, url, parent }) => {
	const token = requireSessionToken(locals);
	const layout =
		typeof parent === 'function' ? await parent() : { ui: DEFAULT_PUBLIC_UI };
	const pageSize = listPageSize(layout.ui);
	const filters = readMusicFilters(url);
	const page = filters.page;
	const offset = (page - 1) * pageSize;
	const currentUrl = `${url.pathname}${url.search}`;
	const api = createApiClient(fetch);

	let albums: MusicAlbum[] = [];
	let artists: MusicArtist[] = [];
	let tracks: MusicTrack[] = [];
	let total = 0;
	let errorMessage: string | null = null;

	try {
		if (filters.view === 'artistas') {
			const response = await api.getMusicArtists(
				{
					limit: pageSize,
					offset,
					q: filters.q || undefined,
					letter: filters.letter ?? undefined
				},
				token
			);
			artists = response.artists ?? [];
			total = response.total ?? 0;
		} else if (filters.view === 'pistas') {
			if (filters.q) {
				const response = await api.getMusicTracks(
					{ limit: pageSize, offset, q: filters.q },
					token
				);
				tracks = response.tracks ?? [];
				total = response.total ?? 0;
			}
		} else {
			const response = await api.getMusicAlbums(
				{
					limit: pageSize,
					offset,
					q: filters.q || undefined,
					letter: filters.letter ?? undefined,
					sort: filters.order
				},
				token
			);
			albums = response.albums ?? [];
			total = response.total ?? 0;
		}
	} catch (cause) {
		if (cause instanceof ApiError && cause.status === 401) {
			redirect(303, '/login');
		}
		errorMessage = LOAD_ERROR;
	}

	const last = pageCount(total, pageSize);
	if (!errorMessage && page > last) {
		redirect(303, pageHref(url, last));
	}

	return {
		filters,
		view: filters.view,
		q: filters.q,
		letter: filters.letter,
		order: filters.order,
		albums,
		artists,
		tracks,
		total,
		page,
		last,
		pageSize,
		currentUrl,
		errorMessage
	};
};
