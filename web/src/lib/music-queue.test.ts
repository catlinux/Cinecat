import { describe, expect, it } from 'vitest';
import {
	appendQueueTrack,
	clampQueueIndex,
	cycleRepeatMode,
	insertQueueTrackAfter,
	nextQueueIndex,
	previousQueueIndex,
	queueIndexForID,
	removeQueueTrack,
	seededRandom,
	shuffleQueue,
	shuffleTracksWithSeed,
	type MusicQueueTrack
} from './music-queue';

function track(id: number): MusicQueueTrack {
	return {
		id,
		title: `Pista ${id}`,
		artist: 'Artista',
		album: 'Álbum',
		albumId: 1,
		durationSeconds: 100 + id
	};
}

const tracks = [track(1), track(2), track(3), track(4)];

describe('índices de la cola', () => {
	it('normaliza índices vacíos y fuera de rango', () => {
		expect(clampQueueIndex(0, 0)).toBe(-1);
		expect(clampQueueIndex(-5, 4)).toBe(0);
		expect(clampQueueIndex(9, 4)).toBe(3);
		expect(clampQueueIndex(Number.NaN, 4)).toBe(0);
	});

	it('avanza con repetir desactivado, todas y una', () => {
		expect(nextQueueIndex(4, 1, 'off')).toBe(2);
		expect(nextQueueIndex(4, 3, 'off')).toBe(-1);
		expect(nextQueueIndex(4, 3, 'all')).toBe(0);
		expect(nextQueueIndex(4, 1, 'one', true)).toBe(1);
		expect(nextQueueIndex(4, 1, 'one', false)).toBe(2);
		expect(nextQueueIndex(0, 0, 'all')).toBe(-1);
	});

	it('retrocede y da la vuelta solo con repetir todas', () => {
		expect(previousQueueIndex(4, 2, 'off')).toBe(1);
		expect(previousQueueIndex(4, 0, 'off')).toBe(0);
		expect(previousQueueIndex(4, 0, 'all')).toBe(3);
		expect(previousQueueIndex(4, 2, 'one')).toBe(1);
		expect(previousQueueIndex(0, 0, 'all')).toBe(-1);
	});
});

describe('operaciones de la cola', () => {
	it('añade al final y después de la pista actual', () => {
		const appended = appendQueueTrack(tracks, track(5));
		expect(appended.map((item) => item.id)).toEqual([1, 2, 3, 4, 5]);
		expect(tracks.map((item) => item.id)).toEqual([1, 2, 3, 4]);

		expect(insertQueueTrackAfter(tracks, 1, track(9)).map((item) => item.id)).toEqual([
			1, 2, 9, 3, 4
		]);
		expect(insertQueueTrackAfter(tracks, -1, track(9)).map((item) => item.id)).toEqual([
			1, 2, 3, 4, 9
		]);
		expect(insertQueueTrackAfter(tracks, 99, track(9)).map((item) => item.id)).toEqual([
			1, 2, 3, 4, 9
		]);
	});

	it('quita una pista anterior y ajusta el índice actual', () => {
		const result = removeQueueTrack(tracks, 2, 0);
		expect(result.tracks.map((item) => item.id)).toEqual([2, 3, 4]);
		expect(result.index).toBe(1);
		expect(result.removedCurrent).toBe(false);
	});

	it('quita una pista posterior sin mover el índice', () => {
		const result = removeQueueTrack(tracks, 1, 3);
		expect(result.tracks.map((item) => item.id)).toEqual([1, 2, 3]);
		expect(result.index).toBe(1);
		expect(result.removedCurrent).toBe(false);
	});

	it('quita la pista actual y deja sonando la siguiente', () => {
		const result = removeQueueTrack(tracks, 1, 1);
		expect(result.tracks.map((item) => item.id)).toEqual([1, 3, 4]);
		expect(result.index).toBe(1);
		expect(result.removedCurrent).toBe(true);
	});

	it('quita la última pista actual y vuelve a la anterior', () => {
		const result = removeQueueTrack(tracks, 3, 3);
		expect(result.tracks.map((item) => item.id)).toEqual([1, 2, 3]);
		expect(result.index).toBe(2);
		expect(result.removedCurrent).toBe(true);
	});

	it('vacía la cola y tolera índices inválidos', () => {
		const result = removeQueueTrack([track(1)], 0, 0);
		expect(result.tracks).toEqual([]);
		expect(result.index).toBe(-1);
		expect(result.removedCurrent).toBe(true);

		const invalid = removeQueueTrack(tracks, 2, 99);
		expect(invalid.tracks).toHaveLength(4);
		expect(invalid.index).toBe(2);
		expect(invalid.removedCurrent).toBe(false);
	});

	it('busca por identificador y cicla repetir', () => {
		expect(queueIndexForID(tracks, 3)).toBe(2);
		expect(queueIndexForID(tracks, 99)).toBe(-1);
		expect(cycleRepeatMode('off')).toBe('all');
		expect(cycleRepeatMode('all')).toBe('one');
		expect(cycleRepeatMode('one')).toBe('off');
	});
});

describe('aleatorio determinista', () => {
	it('genera la misma secuencia con la misma semilla y distinta con otra', () => {
		const first = Array.from({ length: 8 }, seededRandom(42));
		const second = Array.from({ length: 8 }, seededRandom(42));
		const other = Array.from({ length: 8 }, seededRandom(43));
		expect(first).toEqual(second);
		expect(first).not.toEqual(other);
	});

	it('baraja de forma estable toda la lista con una semilla', () => {
		const first = shuffleTracksWithSeed(tracks, 7).map((item) => item.id);
		const second = shuffleTracksWithSeed(tracks, 7).map((item) => item.id);
		expect(first).toEqual(second);
		expect([...first].sort()).toEqual([1, 2, 3, 4]);
	});

	it('no cambia la pista actual ni su posición al barajar', () => {
		const result = shuffleQueue(tracks, 2, 5);
		expect(result.index).toBe(2);
		expect(result.tracks[2].id).toBe(3);
		expect([...result.tracks.map((item) => item.id)].sort()).toEqual([1, 2, 3, 4]);
	});

	it('tolera colas vacías o de una pista', () => {
		expect(shuffleQueue([], -1, 1)).toEqual({ tracks: [], index: -1 });
		expect(shuffleQueue([track(1)], 0, 1)).toEqual({ tracks: [track(1)], index: 0 });
	});
});
