import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { musicPlayer, MUSIC_PLAYER_STORAGE_KEY, pauseMusicForVideo, previousMusicTrack, playQueue, removeFromMusicQueue, restoreMusicPlayer, resetMusicPlayer, setMusicAudioElement, setMusicPlayerFetch, toggleMusicShuffle, toggleMusicPlay, type MusicPlayerState } from './music-player.svelte';
import type { MusicQueueTrack } from './music-queue';

class FakeAudio {
	currentTime = 0;
	duration = 100;
	currentSrc = '';
	src = '';
	volume = 1;
	muted = false;
	private listeners = new Map<string, Set<() => void>>();

	addEventListener(type: string, listener: () => void) {
		if (!this.listeners.has(type)) {
			this.listeners.set(type, new Set());
		}
		this.listeners.get(type)?.add(listener);
	}

	removeEventListener(type: string, listener: () => void) {
		this.listeners.get(type)?.delete(listener);
	}

	dispatch(type: string) {
		for (const listener of this.listeners.get(type) ?? []) {
			listener();
		}
	}

	play() {
		this.currentSrc = this.src;
		this.dispatch('play');
		return Promise.resolve();
	}

	pause() {
		this.dispatch('pause');
	}

	load() {
		this.currentSrc = this.src;
	}

	removeAttribute(name: string) {
		if (name === 'src') {
			this.src = '';
			this.currentSrc = '';
		}
	}
}

function queueTrack(id: number, overrides: Partial<MusicQueueTrack> = {}): MusicQueueTrack {
	return {
		id,
		title: `Pista ${id}`,
		artist: 'Artista',
		artistId: 4,
		album: 'Álbum',
		albumId: 9,
		durationSeconds: 120 + id,
		...overrides
	};
}

function playResponse(url: string, playable = true): Response {
	return new Response(JSON.stringify({ url, mime: 'audio/mpeg', expires_at: '', playable }), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
}

function state(): MusicPlayerState {
	return musicPlayer as MusicPlayerState;
}

describe('reproductor de música', () => {
	let audio: FakeAudio;
	let storage: Map<string, string>;
	let fetchMock: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		resetMusicPlayer();
		audio = new FakeAudio();
		setMusicAudioElement(audio as unknown as HTMLAudioElement);
		storage = new Map();
		vi.stubGlobal('localStorage', {
			getItem: (key: string) => storage.get(key) ?? null,
			setItem: (key: string, value: string) => storage.set(key, value),
			removeItem: (key: string) => storage.delete(key)
		});
		fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const id = Number(String(input).split('/').pop());
			return playResponse(`http://api.test/api/v1/stream/music/${id}?exp=1&sig=x`);
		});
		setMusicPlayerFetch(fetchMock as never);
	});

	afterEach(() => {
		setMusicPlayerFetch(null);
		vi.unstubAllGlobals();
	});

	it('reproduce una cola y pide la URL firmada', async () => {
		playQueue([queueTrack(1), queueTrack(2)], 0);
		await vi.waitFor(() => expect(state().playing).toBe(true));

		expect(fetchMock).toHaveBeenCalledWith('/music/play/1', expect.anything());
		expect(state().tracks).toHaveLength(2);
		expect(state().index).toBe(0);
		expect(audio.src).toContain('/stream/music/1');
		expect(state().duration).toBe(121);
	});

	it('alterna pausa y reproducción', async () => {
		playQueue([queueTrack(1)], 0);
		await vi.waitFor(() => expect(state().playing).toBe(true));

		toggleMusicPlay();
		expect(state().playing).toBe(false);
		toggleMusicPlay();
		await vi.waitFor(() => expect(state().playing).toBe(true));
	});

	it('salta una pista no reproducible sin bloquear la cola', async () => {
		fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
			const id = Number(String(input).split('/').pop());
			return id === 1 ? playResponse('', false) : playResponse(`http://api.test/${id}`);
		});
		playQueue([queueTrack(1), queueTrack(2)], 0);

		await vi.waitFor(() => expect(state().index).toBe(1));
		expect(state().notice).toContain('No se pudo reproducir');
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it('detiene la cola si todas las pistas fallan, aunque repita todas', async () => {
		fetchMock.mockImplementation(async () => playResponse('', false));
		state().repeat = 'all';
		playQueue([queueTrack(1), queueTrack(2), queueTrack(3)], 0);

		await vi.waitFor(() => expect(state().notice).toContain('ninguna pista'));
		expect(fetchMock).toHaveBeenCalledTimes(3);
		expect(state().playing).toBe(false);
	});

	it('reintenta la URL una vez ante un error de audio y después avanza', async () => {
		playQueue([queueTrack(1), queueTrack(2)], 0);
		await vi.waitFor(() => expect(state().playing).toBe(true));
		expect(fetchMock).toHaveBeenCalledTimes(1);

		audio.dispatch('error');
		await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
		expect(state().index).toBe(0);

		audio.dispatch('error');
		await vi.waitFor(() => expect(state().index).toBe(1));
		expect(state().notice).toContain('No se pudo reproducir Pista 1');
	});

	it('reinicia la pista al pulsar anterior después de tres segundos', async () => {
		playQueue([queueTrack(1), queueTrack(2)], 1);
		await vi.waitFor(() => expect(state().playing).toBe(true));
		audio.currentTime = 5;

		previousMusicTrack();
		expect(state().index).toBe(1);
		expect(audio.currentTime).toBe(0);
		expect(state().currentTime).toBe(0);
	});

	it('quitar la pista actual empieza la siguiente', async () => {
		playQueue([queueTrack(1), queueTrack(2), queueTrack(3)], 1);
		await vi.waitFor(() => expect(state().playing).toBe(true));

		removeFromMusicQueue(1);
		await vi.waitFor(() => expect(state().tracks[1]?.id).toBe(3));
		expect(state().index).toBe(1);
	});

	it('baraja sin cambiar la pista actual', async () => {
		playQueue([queueTrack(1), queueTrack(2), queueTrack(3), queueTrack(4)], 2);
		await vi.waitFor(() => expect(state().playing).toBe(true));

		toggleMusicShuffle();
		expect(state().shuffle).toBe(true);
		expect(state().index).toBe(2);
		expect(state().tracks[2].id).toBe(3);
	});

	it('persiste y restaura la cola en pausa', async () => {
		playQueue([queueTrack(1), queueTrack(2)], 1);
		await vi.waitFor(() => expect(state().playing).toBe(true));
		expect(storage.get(MUSIC_PLAYER_STORAGE_KEY)).toBeTruthy();

		audio.pause();
		setMusicAudioElement(null);
		resetMusicPlayer();
		restoreMusicPlayer();

		expect(state().tracks.map((track) => track.id)).toEqual([1, 2]);
		expect(state().index).toBe(1);
		expect(state().playing).toBe(false);
		expect(fetchMock).toHaveBeenCalledTimes(1);
		setMusicAudioElement(audio as unknown as HTMLAudioElement);
	});

	it('pausa la música cuando empieza un vídeo sin reanudarla', () => {
		musicPlayer.playing = true;
		expect(pauseMusicForVideo({ tagName: 'VIDEO' } as unknown as EventTarget)).toBe(true);
		expect(state().playing).toBe(false);
		expect(pauseMusicForVideo({ tagName: 'DIV' } as unknown as EventTarget)).toBe(false);
	});

	it('tolera localStorage ausente o corrupto', () => {
		vi.stubGlobal('localStorage', undefined);
		resetMusicPlayer();
		restoreMusicPlayer();
		expect(state().tracks).toEqual([]);

		storage.set(MUSIC_PLAYER_STORAGE_KEY, '{roto');
		resetMusicPlayer();
		restoreMusicPlayer();
		expect(state().tracks).toEqual([]);

		storage.set(MUSIC_PLAYER_STORAGE_KEY, 'null');
		resetMusicPlayer();
		restoreMusicPlayer();
		expect(state().tracks).toEqual([]);
	});
});
