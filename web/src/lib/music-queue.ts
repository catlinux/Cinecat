export type MusicRepeatMode = 'off' | 'all' | 'one';

export interface MusicQueueTrack {
	id: number;
	title: string;
	artist: string;
	artistId?: number;
	album: string;
	albumId: number;
	durationSeconds: number;
}

export interface QueueRemoval {
	tracks: MusicQueueTrack[];
	index: number;
	removedCurrent: boolean;
}

export interface ShuffledQueue {
	tracks: MusicQueueTrack[];
	index: number;
}

export function clampQueueIndex(index: number, length: number): number {
	if (!Number.isFinite(length) || length <= 0) {
		return -1;
	}
	if (!Number.isFinite(index)) {
		return 0;
	}
	return Math.min(length - 1, Math.max(0, Math.trunc(index)));
}

// auto distingue el final natural de una pista del botón «siguiente»: con
// repetir una, el final repite, pero el botón avanza.
export function nextQueueIndex(
	length: number,
	index: number,
	repeat: MusicRepeatMode,
	auto = true
): number {
	if (!Number.isFinite(length) || length <= 0) {
		return -1;
	}
	if (index < 0) {
		return 0;
	}
	if (repeat === 'one' && auto) {
		return clampQueueIndex(index, length);
	}
	if (index + 1 < length) {
		return index + 1;
	}
	return repeat === 'all' ? 0 : -1;
}

export function previousQueueIndex(
	length: number,
	index: number,
	repeat: MusicRepeatMode
): number {
	if (!Number.isFinite(length) || length <= 0) {
		return -1;
	}
	if (index <= 0) {
		return repeat === 'all' ? length - 1 : 0;
	}
	return clampQueueIndex(index - 1, length);
}

export function appendQueueTrack(
	tracks: MusicQueueTrack[],
	track: MusicQueueTrack
): MusicQueueTrack[] {
	return [...tracks, track];
}

export function insertQueueTrackAfter(
	tracks: MusicQueueTrack[],
	index: number,
	track: MusicQueueTrack
): MusicQueueTrack[] {
	const result = [...tracks];
	const at = index < 0 ? result.length : Math.min(result.length, index + 1);
	result.splice(at, 0, track);
	return result;
}

export function removeQueueTrack(
	tracks: MusicQueueTrack[],
	index: number,
	removeIndex: number
): QueueRemoval {
	if (removeIndex < 0 || removeIndex >= tracks.length) {
		return { tracks: [...tracks], index: clampQueueIndex(index, tracks.length), removedCurrent: false };
	}
	const next = tracks.filter((_, current) => current !== removeIndex);
	if (next.length === 0) {
		return { tracks: next, index: -1, removedCurrent: removeIndex === index };
	}
	if (removeIndex < index) {
		return { tracks: next, index: index - 1, removedCurrent: false };
	}
	if (removeIndex > index) {
		return { tracks: next, index, removedCurrent: false };
	}
	return {
		tracks: next,
		index: Math.min(index, next.length - 1),
		removedCurrent: true
	};
}

export function queueIndexForID(tracks: MusicQueueTrack[], id: number): number {
	return tracks.findIndex((track) => track.id === id);
}

export function cycleRepeatMode(mode: MusicRepeatMode): MusicRepeatMode {
	if (mode === 'off') {
		return 'all';
	}
	if (mode === 'all') {
		return 'one';
	}
	return 'off';
}

export function seededRandom(seed: number): () => number {
	let state = Number.isFinite(seed) ? Math.trunc(seed) >>> 0 : 0;
	return () => {
		state = (state + 0x6d2b79f5) >>> 0;
		let value = state;
		value = Math.imul(value ^ (value >>> 15), value | 1);
		value ^= value + Math.imul(value ^ (value >>> 7), value | 61);
		return ((value ^ (value >>> 14)) >>> 0) / 4294967296;
	};
}

export function shuffleTracksWithSeed(
	tracks: MusicQueueTrack[],
	seed = 1
): MusicQueueTrack[] {
	const result = [...tracks];
	const random = seededRandom(seed);
	for (let index = result.length - 1; index > 0; index--) {
		const swap = Math.floor(random() * (index + 1));
		[result[index], result[swap]] = [result[swap], result[index]];
	}
	return result;
}

// Baraja el resto de la cola y deja la pista actual en su misma posición para
// que el botón aleatorio no cambie lo que está sonando.
export function shuffleQueue(
	tracks: MusicQueueTrack[],
	index: number,
	seed = 1
): ShuffledQueue {
	if (tracks.length <= 1) {
		return { tracks: [...tracks], index: clampQueueIndex(index, tracks.length) };
	}
	const at = clampQueueIndex(index, tracks.length);
	const current = tracks[at];
	if (!current) {
		return { tracks: shuffleTracksWithSeed(tracks, seed), index: tracks.length > 0 ? 0 : -1 };
	}
	const positions = tracks.map((_, position) => position).filter((position) => position !== at);
	const random = seededRandom(seed);
	for (let position = positions.length - 1; position > 0; position--) {
		const swap = Math.floor(random() * (position + 1));
		[positions[position], positions[swap]] = [positions[swap], positions[position]];
	}
	const shuffled = positions.map((position) => tracks[position]);
	shuffled.splice(at, 0, current);
	return { tracks: shuffled, index: at };
}

export const insertAfter = insertQueueTrackAfter;
export const removeAt = removeQueueTrack;
export const shuffle = shuffleQueue;
export const nextIndex = nextQueueIndex;
export const previousIndex = previousQueueIndex;
export const shuffleTracks = shuffleTracksWithSeed;
