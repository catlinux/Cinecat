import { error, redirect } from '@sveltejs/kit';
import { ApiError, type MusicArtistDetail } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { parseMusicID, toQueueTrack } from '$lib/music';
import type { MusicQueueTrack } from '$lib/music-queue';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, locals, fetch }) => {
	const token = requireSessionToken(locals);
	const id = parseMusicID(params.id);
	if (id === null) {
		error(404, 'Artista no encontrado');
	}

	const api = createApiClient(fetch);
	let artist: MusicArtistDetail;
	try {
		artist = await api.getMusicArtist(id, { limit: 200, offset: 0 }, token);
	} catch (cause) {
		if (cause instanceof ApiError) {
			if (cause.status === 401) {
				redirect(303, '/login');
			}
			if (cause.status === 404) {
				error(404, 'Artista no encontrado');
			}
		}
		error(502, 'No se pudo cargar el artista.');
	}

	const queueTracks: MusicQueueTrack[] = [];
	for (const album of artist.albums ?? []) {
		try {
			const detail = await api.getMusicAlbum(album.id, { limit: 200, offset: 0 }, token);
			for (const track of detail.tracks ?? []) {
				if (track.playable) {
					queueTracks.push(toQueueTrack(track, detail));
				}
			}
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 401) {
				redirect(303, '/login');
			}
			// Un álbum roto no impide ver el resto del artista.
		}
	}

	return { artist, albums: artist.albums ?? [], queueTracks };
};
