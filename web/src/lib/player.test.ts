import { describe, expect, it } from 'vitest';
import {
	clampSeekTime,
	controlsShouldHide,
	countdownLabel,
	cycleSpeed,
	formatPlayerTime,
	formatSpeed,
	pointerPosition,
	shortcutAction
} from './player';

describe('utilidades del reproductor', () => {
	it('formatea tiempos con horas, minutos y segundos', () => {
		expect(formatPlayerTime(0)).toBe('0:00:00');
		expect(formatPlayerTime(65)).toBe('0:01:05');
		expect(formatPlayerTime(3723)).toBe('1:02:03');
		expect(formatPlayerTime(Number.NaN)).toBe('0:00:00');
	});

	it('calcula la posición al pulsar la barra y la limita', () => {
		expect(pointerPosition(150, 100, 200, 600)).toBe(150);
		expect(pointerPosition(5, 100, 200, 600)).toBe(0);
		expect(pointerPosition(400, 100, 200, 600)).toBe(600);
		expect(clampSeekTime(50, -10, 100)).toBe(40);
		expect(clampSeekTime(5, -10, 100)).toBe(0);
		expect(clampSeekTime(95, 20, 100)).toBe(100);
	});

	it('cicla las velocidades', () => {
		expect(cycleSpeed(1)).toBe(1.25);
		expect(cycleSpeed(2)).toBe(0.5);
		expect(cycleSpeed(0.5, -1)).toBe(2);
		expect(formatSpeed(0.5)).toBe('0,5×');
	});

	it('oculta los controles solo al reproducir y sin interacción', () => {
		expect(controlsShouldHide(true, false, false, 0, 3001)).toBe(true);
		expect(controlsShouldHide(false, false, false, 0, 5000)).toBe(false);
		expect(controlsShouldHide(true, true, false, 0, 5000)).toBe(false);
		expect(controlsShouldHide(true, false, true, 0, 5000)).toBe(false);
	});

	it('mapea los atajos a acciones', () => {
		const base = { paused: false, duration: 100, currentTime: 20, volume: 0.5 };
		expect(shortcutAction({ ...base, key: ' ' })).toEqual({ type: 'toggle' });
		expect(shortcutAction({ ...base, key: 'ArrowLeft' })).toEqual({ type: 'seek', value: 10 });
		expect(shortcutAction({ ...base, key: 'ArrowRight', shiftKey: true })).toEqual({
			type: 'seek',
			value: 50
		});
		expect(shortcutAction({ ...base, key: 'ArrowUp' })).toEqual({ type: 'volume', value: 0.55 });
		expect(shortcutAction({ ...base, key: 'm' })).toEqual({ type: 'mute' });
		expect(shortcutAction({ ...base, key: 'f' })).toEqual({ type: 'fullscreen' });
		expect(shortcutAction({ ...base, key: 'c' })).toEqual({ type: 'subtitles' });
		expect(shortcutAction({ ...base, key: ',', paused: true })).toEqual({
			type: 'frame',
			delta: -1 / 24
		});
		expect(shortcutAction({ ...base, key: ',', paused: false })).toBeNull();
		expect(shortcutAction({ ...base, key: '5', duration: 100 })).toEqual({
			type: 'percent',
			value: 50
		});
		expect(shortcutAction({ ...base, key: 'ArrowRight', ctrlKey: true })).toBeNull();
	});

	it('muestra la cuenta atrás del siguiente episodio', () => {
		expect(countdownLabel(8)).toBe('Siguiente episodio en 8…');
	});
});
