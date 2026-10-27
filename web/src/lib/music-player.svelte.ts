import {
	appendQueueTrack,
	clampQueueIndex,
	cycleRepeatMode,
	insertQueueTrackAfter,
	nextQueueIndex,
	previousQueueIndex,
	queueIndexForID,
	removeQueueTrack,
	shuffleQueue,
	shuffleTracksWithSeed,
	type MusicQueueTrack,
	type MusicRepeatMode
} from './music-queue';

export const MUSIC_PLAYER_STORAGE_KEY = 'cinecat.music.player';
const PLAYER_VOLUME_EPSILON = 0.001;
const PREVIOUS_RESTART_SECONDS = 3;

type MusicPlayerFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

export interface MusicPlayerState {
	tracks: MusicQueueTrack[];
	index: number;
	playing: boolean;
	currentTime: number;
	duration: number;
	volume: number;
	muted: boolean;
	shuffle: boolean;
	repeat: MusicRepeatMode;
	notice: string | null;
}

interface StoredPlayerState {
	tracks?: unknown;
	index?: unknown;
	volume?: unknown;
	muted?: unknown;
	shuffle?: unknown;
	repeat?: unknown;
}

interface MusicPlayJSON {
	url: string;
	playable: boolean;
}

let audioElement: HTMLAudioElement | null = null;
let audioCleanup: (() => void) | null = null;
let loadSequence = 0;
let retriedCurrent = false;
// Fallos seguidos sin ninguna pista que haya sonado: con «repetir todas» y la
// cola entera fallando, saltar de pista en pista no terminaría nunca.
let consecutiveFailures = 0;
let restored = false;
let fetchImpl: MusicPlayerFetch | null = null;

export const musicPlayer = $state<MusicPlayerState>({
	tracks: [],
	index: -1,
	playing: false,
	currentTime: 0,
	duration: 0,
	volume: 1,
	muted: false,
	shuffle: false,
	repeat: 'off',
	notice: null
});

export function currentMusicTrack(): MusicQueueTrack | null {
	return musicPlayer.index >= 0 && musicPlayer.index < musicPlayer.tracks.length
		? musicPlayer.tracks[musicPlayer.index]
		: null;
}

export function setMusicPlayerFetch(fetchFunction: MusicPlayerFetch | null): void {
	fetchImpl = fetchFunction;
}

export function setMusicAudioElement(element: HTMLAudioElement | null): void {
	if (audioCleanup) {
		audioCleanup();
		audioCleanup = null;
	}
	audioElement = element;
	if (!element) {
		return;
	}
	element.volume = musicPlayer.volume;
	element.muted = musicPlayer.muted;
	const onTime = () => {
		musicPlayer.currentTime = Number.isFinite(element.currentTime) ? element.currentTime : 0;
	};
	const onDuration = () => {
		const duration = Number.isFinite(element.duration) && element.duration > 0
			? element.duration
			: currentMusicTrack()?.durationSeconds ?? 0;
		musicPlayer.duration = duration;
	};
	const onPlay = () => {
		musicPlayer.playing = true;
		consecutiveFailures = 0;
	};
	const onPause = () => {
		musicPlayer.playing = false;
	};
	const onEnded = () => {
		void handleTrackEnded();
	};
	const onError = () => {
		void handleAudioFailure();
	};
	element.addEventListener('timeupdate', onTime);
	element.addEventListener('durationchange', onDuration);
	element.addEventListener('loadedmetadata', onDuration);
	element.addEventListener('play', onPlay);
	element.addEventListener('pause', onPause);
	element.addEventListener('ended', onEnded);
	element.addEventListener('error', onError);
	audioCleanup = () => {
		element.removeEventListener('timeupdate', onTime);
		element.removeEventListener('durationchange', onDuration);
		element.removeEventListener('loadedmetadata', onDuration);
		element.removeEventListener('play', onPlay);
		element.removeEventListener('pause', onPause);
		element.removeEventListener('ended', onEnded);
		element.removeEventListener('error', onError);
	};
}

export function playQueue(
	tracks: MusicQueueTrack[],
	startIndex = 0,
	options: { shuffle?: boolean } = {}
): void {
	const queue = tracks.filter((track) => track.id > 0);
	if (queue.length === 0) {
		clearMusicQueue();
		return;
	}
	let nextTracks = queue;
	let index = clampQueueIndex(startIndex, queue.length);
	if (options.shuffle) {
		nextTracks = shuffleTracksWithSeed(queue, Date.now());
		index = 0;
		musicPlayer.shuffle = true;
	}
	musicPlayer.tracks = nextTracks;
	musicPlayer.index = index;
	musicPlayer.currentTime = 0;
	musicPlayer.duration = nextTracks[index]?.durationSeconds ?? 0;
	musicPlayer.notice = null;
	retriedCurrent = false;
	consecutiveFailures = 0;
	persistMusicPlayer();
	void loadAndPlay(true);
}

export function playTrack(track: MusicQueueTrack, tracks: MusicQueueTrack[] = []): void {
	const queue = tracks.length > 0 ? tracks : musicPlayer.tracks;
	const index = queue.findIndex((item) => item.id === track.id);
	if (queue.length === 0 || index < 0) {
		playQueue([track], 0);
		return;
	}
	void startQueueAt(queue, index);
}

async function startQueueAt(tracks: MusicQueueTrack[], index: number): Promise<void> {
	musicPlayer.tracks = [...tracks];
	musicPlayer.index = clampQueueIndex(index, tracks.length);
	musicPlayer.currentTime = 0;
	musicPlayer.duration = currentMusicTrack()?.durationSeconds ?? 0;
	musicPlayer.notice = null;
	retriedCurrent = false;
	consecutiveFailures = 0;
	persistMusicPlayer();
	await loadAndPlay(true);
}

export function appendToMusicQueue(track: MusicQueueTrack): void {
	musicPlayer.tracks = appendQueueTrack(musicPlayer.tracks, track);
	if (musicPlayer.index < 0) {
		musicPlayer.index = 0;
		musicPlayer.duration = track.durationSeconds;
	}
	persistMusicPlayer();
}

export function appendManyToMusicQueue(tracks: MusicQueueTrack[]): void {
	for (const track of tracks) {
		if (track.id > 0) {
			musicPlayer.tracks = appendQueueTrack(musicPlayer.tracks, track);
		}
	}
	if (musicPlayer.index < 0 && musicPlayer.tracks.length > 0) {
		musicPlayer.index = 0;
		musicPlayer.duration = musicPlayer.tracks[0].durationSeconds;
	}
	persistMusicPlayer();
}

export function playNextInMusicQueue(track: MusicQueueTrack): void {
	if (musicPlayer.index < 0) {
		playQueue([track], 0);
		return;
	}
	musicPlayer.tracks = insertQueueTrackAfter(musicPlayer.tracks, musicPlayer.index, track);
	persistMusicPlayer();
}

export function removeFromMusicQueue(index: number): void {
	const wasCurrent = index === musicPlayer.index;
	const wasPlaying = musicPlayer.playing;
	const result = removeQueueTrack(musicPlayer.tracks, musicPlayer.index, index);
	musicPlayer.tracks = result.tracks;
	musicPlayer.index = result.index;
	if (musicPlayer.tracks.length === 0) {
		clearMusicQueue();
		return;
	}
	persistMusicPlayer();
	if (wasCurrent) {
		musicPlayer.currentTime = 0;
		musicPlayer.duration = currentMusicTrack()?.durationSeconds ?? 0;
		retriedCurrent = false;
		void loadAndPlay(wasPlaying);
	}
}

export function clearMusicQueue(): void {
	musicPlayer.tracks = [];
	musicPlayer.index = -1;
	musicPlayer.playing = false;
	musicPlayer.currentTime = 0;
	musicPlayer.duration = 0;
	musicPlayer.notice = null;
	if (audioElement) {
		audioElement.pause();
		audioElement.removeAttribute('src');
		audioElement.load();
	}
	persistMusicPlayer();
}

export function toggleMusicPlay(): void {
	const track = currentMusicTrack();
	if (!track) {
		if (musicPlayer.tracks.length > 0) {
			musicPlayer.index = 0;
			void loadAndPlay(true);
		}
		return;
	}
	if (musicPlayer.playing) {
		pauseMusic();
		return;
	}
	if (!audioElement || !audioElement.currentSrc) {
		void loadAndPlay(true);
		return;
	}
	void audioElement.play().catch(() => {
		musicPlayer.playing = false;
	});
}

export function pauseMusic(): void {
	if (audioElement) {
		audioElement.pause();
	}
	musicPlayer.playing = false;
}

export function resumeMusic(): void {
	const track = currentMusicTrack();
	if (!track) {
		return;
	}
	if (audioElement?.currentSrc) {
		void audioElement.play().catch(() => {
			musicPlayer.playing = false;
		});
		return;
	}
	void loadAndPlay(true);
}

export function nextMusicTrack(auto = false): void {
	const next = nextQueueIndex(
		musicPlayer.tracks.length,
		musicPlayer.index,
		musicPlayer.repeat,
		auto
	);
	if (next < 0) {
		musicPlayer.playing = false;
		audioElement?.pause();
		return;
	}
	musicPlayer.index = next;
	musicPlayer.currentTime = 0;
	musicPlayer.duration = currentMusicTrack()?.durationSeconds ?? 0;
	retriedCurrent = false;
	musicPlayer.notice = null;
	persistMusicPlayer();
	void loadAndPlay(true);
}

export function previousMusicTrack(): void {
	if (audioElement && audioElement.currentTime > PREVIOUS_RESTART_SECONDS) {
		seekMusic(0);
		return;
	}
	const previous = previousQueueIndex(
		musicPlayer.tracks.length,
		musicPlayer.index,
		musicPlayer.repeat
	);
	if (previous < 0) {
		return;
	}
	musicPlayer.index = previous;
	musicPlayer.currentTime = 0;
	musicPlayer.duration = currentMusicTrack()?.durationSeconds ?? 0;
	retriedCurrent = false;
	musicPlayer.notice = null;
	persistMusicPlayer();
	void loadAndPlay(true);
}

export function seekMusic(seconds: number): void {
	const duration = musicPlayer.duration > 0 ? musicPlayer.duration : audioElement?.duration ?? 0;
	const value = Math.max(0, Math.min(duration || 0, Number.isFinite(seconds) ? seconds : 0));
	musicPlayer.currentTime = value;
	if (audioElement) {
		try {
			audioElement.currentTime = value;
		} catch {
			// Algunos navegadores rechazan el salto antes de cargar metadatos.
		}
	}
}

export function setMusicVolume(volume: number): void {
	const value = Math.max(0, Math.min(1, Number.isFinite(volume) ? volume : 1));
	musicPlayer.volume = value;
	musicPlayer.muted = false;
	if (audioElement) {
		audioElement.volume = value;
		audioElement.muted = false;
	}
	persistMusicPlayer();
}

export function toggleMusicMute(): void {
	musicPlayer.muted = !musicPlayer.muted;
	if (audioElement) {
		audioElement.muted = musicPlayer.muted;
	}
	persistMusicPlayer();
}

export function toggleMusicShuffle(): void {
	musicPlayer.shuffle = !musicPlayer.shuffle;
	if (musicPlayer.shuffle && musicPlayer.tracks.length > 1) {
		const shuffled = shuffleQueue(musicPlayer.tracks, musicPlayer.index, Date.now());
		musicPlayer.tracks = shuffled.tracks;
		musicPlayer.index = shuffled.index;
	}
	persistMusicPlayer();
}

export function cycleMusicRepeat(): void {
	musicPlayer.repeat = cycleRepeatMode(musicPlayer.repeat);
	persistMusicPlayer();
}

export function setMusicShuffle(enabled: boolean): void {
	if (enabled === musicPlayer.shuffle) {
		return;
	}
	toggleMusicShuffle();
}

export function restoreMusicPlayer(): void {
	if (restored) {
		return;
	}
	restored = true;
	const storage = musicStorage();
	if (!storage) {
		return;
	}
	let parsed: StoredPlayerState;
	try {
		const raw = storage.getItem(MUSIC_PLAYER_STORAGE_KEY);
		if (!raw) {
			return;
		}
		parsed = JSON.parse(raw) as StoredPlayerState;
	} catch {
		return;
	}
	if (!parsed || typeof parsed !== 'object') {
		return;
	}
	const tracks = parseStoredTracks(parsed.tracks);
	if (tracks.length === 0) {
		return;
	}
	musicPlayer.tracks = tracks;
	musicPlayer.index = clampQueueIndex(Number(parsed.index), tracks.length);
	musicPlayer.volume = clampVolume(parsed.volume, 1);
	musicPlayer.muted = parsed.muted === true;
	musicPlayer.shuffle = parsed.shuffle === true;
	musicPlayer.repeat =
		parsed.repeat === 'all' || parsed.repeat === 'one' || parsed.repeat === 'off'
			? parsed.repeat
			: 'off';
	musicPlayer.playing = false;
	musicPlayer.currentTime = 0;
	musicPlayer.duration = currentMusicTrack()?.durationSeconds ?? 0;
	if (audioElement) {
		audioElement.volume = musicPlayer.volume;
		audioElement.muted = musicPlayer.muted;
	}
}

export function resetMusicPlayer(): void {
	loadSequence++;
	retriedCurrent = false;
	restored = false;
	musicPlayer.tracks = [];
	musicPlayer.index = -1;
	musicPlayer.playing = false;
	musicPlayer.currentTime = 0;
	musicPlayer.duration = 0;
	musicPlayer.volume = 1;
	musicPlayer.muted = false;
	musicPlayer.shuffle = false;
	musicPlayer.repeat = 'off';
	musicPlayer.notice = null;
	if (audioElement) {
		audioElement.pause();
		audioElement.removeAttribute('src');
	}
}

export function notifyMusicPlayer(message: string | null): void {
	musicPlayer.notice = message;
}

// La reproducción de vídeo manda: si empieza un <video>, la música se pausa y
// no se reanuda por su cuenta al terminar el vídeo.
export function pauseMusicForVideo(target: EventTarget | null): boolean {
	const node = target as { tagName?: string } | null;
	if (node?.tagName?.toUpperCase() !== 'VIDEO') {
		return false;
	}
	pauseMusic();
	return true;
}

async function loadAndPlay(autoplay: boolean): Promise<void> {
	const track = currentMusicTrack();
	if (!track || !audioElement) {
		return;
	}
	const sequence = ++loadSequence;
	const fetchFunction = fetchImpl ?? globalThis.fetch;
	if (!fetchFunction) {
		await failCurrentTrack('No se pudo conectar con el servidor.');
		return;
	}
	let response: Response;
	try {
		response = await fetchFunction(`/music/play/${track.id}`, {
			headers: { Accept: 'application/json' },
			cache: 'no-store'
		});
	} catch {
		if (sequence !== loadSequence) {
			return;
		}
		await failCurrentTrack('No se pudo conectar con el servidor.');
		return;
	}
	if (sequence !== loadSequence) {
		return;
	}
	let payload: MusicPlayJSON;
	try {
		payload = (await response.json()) as MusicPlayJSON;
	} catch {
		await failCurrentTrack('El servidor devolvió una respuesta no válida.');
		return;
	}
	if (payload.playable === false) {
		await failCurrentTrack('Formato no compatible con el navegador', false);
		return;
	}
	if (!response.ok || !payload.url) {
		await failCurrentTrack('No se pudo preparar la reproducción');
		return;
	}
	audioElement.src = payload.url;
	try {
		audioElement.load();
	} catch {
		// Algunos dobles de prueba no implementan load; play seguirá funcionando.
	}
	if (!autoplay) {
		return;
	}
	try {
		await audioElement.play();
	} catch {
		if (sequence !== loadSequence) {
			return;
		}
		musicPlayer.playing = false;
	}
}

async function failCurrentTrack(reason: string, retryable = true): Promise<void> {
	const track = currentMusicTrack();
	if (!track) {
		return;
	}
	if (retryable && !retriedCurrent) {
		retriedCurrent = true;
		await loadAndPlay(true);
		return;
	}
	musicPlayer.notice = `No se pudo reproducir ${track.title}`;
	musicPlayer.playing = false;
	skipAfterFailure();
}

function skipAfterFailure(): void {
	consecutiveFailures++;
	if (consecutiveFailures >= musicPlayer.tracks.length) {
		musicPlayer.notice = 'No se pudo reproducir ninguna pista de la cola';
		musicPlayer.playing = false;
		consecutiveFailures = 0;
		return;
	}
	const repeat = musicPlayer.repeat === 'one' ? 'off' : musicPlayer.repeat;
	const next = nextQueueIndex(musicPlayer.tracks.length, musicPlayer.index, repeat, true);
	if (next < 0) {
		return;
	}
	musicPlayer.index = next;
	musicPlayer.currentTime = 0;
	musicPlayer.duration = currentMusicTrack()?.durationSeconds ?? 0;
	retriedCurrent = false;
	persistMusicPlayer();
	void loadAndPlay(true);
}

async function handleAudioFailure(): Promise<void> {
	await failCurrentTrack('Error de reproducción');
}

async function handleTrackEnded(): Promise<void> {
	if (musicPlayer.repeat === 'one') {
		if (audioElement) {
			audioElement.currentTime = 0;
			try {
				await audioElement.play();
				return;
			} catch {
				// Si el navegador rechaza repetir, se avanza como respaldo.
			}
		}
	}
	nextMusicTrack(true);
}

function clampVolume(value: unknown, fallback: number): number {
	const number = Number(value);
	return Number.isFinite(number) ? Math.max(0, Math.min(1, number)) : fallback;
}

function parseStoredTracks(value: unknown): MusicQueueTrack[] {
	if (!Array.isArray(value)) {
		return [];
	}
	const tracks: MusicQueueTrack[] = [];
	for (const item of value) {
		if (!item || typeof item !== 'object') {
			continue;
		}
		const record = item as Record<string, unknown>;
		const id = Number(record.id);
		if (!Number.isInteger(id) || id <= 0) {
			continue;
		}
		tracks.push({
			id,
			title: typeof record.title === 'string' ? record.title : 'Pista',
			artist: typeof record.artist === 'string' ? record.artist : '',
			artistId: Number.isInteger(Number(record.artistId)) ? Number(record.artistId) : undefined,
			album: typeof record.album === 'string' ? record.album : '',
			albumId: Number.isInteger(Number(record.albumId)) ? Number(record.albumId) : 0,
			durationSeconds: Number.isFinite(Number(record.durationSeconds))
				? Number(record.durationSeconds)
				: 0
		});
	}
	return tracks;
}

function musicStorage(): Storage | null {
	if (typeof localStorage === 'undefined') {
		return null;
	}
	try {
		return localStorage;
	} catch {
		return null;
	}
}

function persistMusicPlayer(): void {
	const storage = musicStorage();
	if (!storage) {
		return;
	}
	try {
		storage.setItem(
			MUSIC_PLAYER_STORAGE_KEY,
			JSON.stringify({
				tracks: musicPlayer.tracks.map((track) => ({
					id: track.id,
					title: track.title,
					artist: track.artist,
					artistId: track.artistId,
					album: track.album,
					albumId: track.albumId,
					durationSeconds: track.durationSeconds
				})),
				index: musicPlayer.index,
				volume: musicPlayer.volume,
				muted: musicPlayer.muted,
				shuffle: musicPlayer.shuffle,
				repeat: musicPlayer.repeat
			})
		);
	} catch {
		// localStorage lleno o bloqueado: el reproductor sigue funcionando.
	}
}

export function musicQueueTrackForID(id: number): MusicQueueTrack | null {
	const index = queueIndexForID(musicPlayer.tracks, id);
	return index >= 0 ? musicPlayer.tracks[index] : null;
}

// Alias breves para las acciones del reproductor; los nombres largos se usan
// dentro de la aplicación para evitar ambigüedad con el reproductor de vídeo.
export const playTracks = playQueue;
export const addToQueue = appendToMusicQueue;
export const addManyToQueue = appendManyToMusicQueue;
export const playNext = playNextInMusicQueue;
export const removeTrack = removeFromMusicQueue;
export const clearQueue = clearMusicQueue;
export const toggle = toggleMusicPlay;
export const next = nextMusicTrack;
export const previous = previousMusicTrack;
export const seek = seekMusic;
export const setVolume = setMusicVolume;
export const toggleMute = toggleMusicMute;
export const toggleShuffle = toggleMusicShuffle;
export const cycleRepeat = cycleMusicRepeat;
export const pauseForVideo = pauseMusicForVideo;
