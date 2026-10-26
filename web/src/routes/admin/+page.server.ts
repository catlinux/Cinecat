import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import type { ScanStatus, ServerVersion } from '$lib/api';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import { webVersion } from '$lib/version';

async function optional<T>(work: () => Promise<T>): Promise<T | null> {
	try {
		return await work();
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		return null;
	}
}

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const token = requireSessionToken(locals);
	const api = createApiClient(fetch);

	const [movies, unmatchedMovies, series, unmatchedSeries, collections, scan, serverInfo] =
		await Promise.all([
			optional(async () => (await api.getItems({ kind: 'movie', limit: 1 }, token)).total),
			optional(
				async () =>
					(await api.getItems({ kind: 'movie', limit: 1, unmatched: true }, token)).total
			),
			optional(async () => (await api.getSeries({ limit: 1 }, token)).total),
			optional(async () => (await api.getSeries({ limit: 1, unmatched: true }, token)).total),
			optional(async () => (await api.getCollections({ limit: 1 }, token)).total),
			optional(async () => api.getScanStatus(token) as Promise<ScanStatus>),
			optional(async () => api.getServerVersion() as Promise<ServerVersion>)
		]);

	return {
		counts: {
			movies,
			unmatchedMovies,
			series,
			unmatchedSeries,
			collections
		},
		scan,
		serverInfo,
		webVersion
	};
};
