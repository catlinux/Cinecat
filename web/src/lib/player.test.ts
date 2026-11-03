import { describe, expect, it } from 'vitest';
import {
	activeSegmentButton,
	clampSeekTime,
	controlsShouldHide,
	countdownLabel,
	cycleSpeed,
	firstSubtitleForToggle,
	formatPlayerTime,
	formatSpeed,
	playbackReloadURL,
	pointerPosition,
	shortcutAction,
	shouldStartAutoplayCountdown,
	subtitleNeedsReload
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

	it('sabe cuándo hay que recargar por un subtítulo de imagen', () => {
		const subtitles = [
			{ id: 'e0', burn_in: false },
			{ id: 'e1', burn_in: true }
		];
		expect(subtitleNeedsReload('e0', 'e1', subtitles)).toBe(true);
		expect(subtitleNeedsReload('e1', 'e0', subtitles)).toBe(true);
		expect(subtitleNeedsReload('e1', null, subtitles)).toBe(true);
		expect(subtitleNeedsReload('e0', null, subtitles)).toBe(false);
		expect(subtitleNeedsReload('e1', 'e1', subtitles)).toBe(false);
	});

	it('elige la primera pista de texto y recurre a la imagen si no hay otra', () => {
		expect(
			firstSubtitleForToggle([
				{ id: 'e0', burn_in: true },
				{ id: 'e1', burn_in: false }
			])
		).toBe('e1');
		expect(firstSubtitleForToggle([{ id: 'e0', burn_in: true }])).toBe('e0');
		expect(firstSubtitleForToggle([])).toBeNull();
	});

	it('decide el botón de segmento según el tiempo actual', () => {
		const segments = [
			{ kind: 'intro' as const, start: 10, end: 20, source: 'chapter' as const },
			{ kind: 'credits' as const, start: 80, end: 90, source: 'manual' as const }
		];
		expect(activeSegmentButton(9.99, segments, false)).toBeNull();
		expect(activeSegmentButton(10, segments, false)).toEqual({
			kind: 'intro',
			label: 'Saltar intro',
			action: 'seek',
			end: 20
		});
		expect(activeSegmentButton(19.99, segments, false)?.action).toBe('seek');
		expect(activeSegmentButton(20, segments, false)).toBeNull();
		expect(activeSegmentButton(80, segments, true)).toEqual({
			kind: 'credits',
			label: 'Siguiente episodio',
			action: 'next',
			end: 90
		});
		expect(activeSegmentButton(80, segments, false)).toEqual({
			kind: 'credits',
			label: 'Saltar créditos',
			action: 'seek',
			end: 90
		});
		expect(activeSegmentButton(Number.NaN, segments, true)).toBeNull();
		expect(activeSegmentButton(10, undefined, true)).toBeNull();
	});

	it('decide si arranca la cuenta atras al terminar', () => {
		expect(shouldStartAutoplayCountdown(true, true, true)).toBe(true);
		expect(shouldStartAutoplayCountdown(undefined, true, true)).toBe(true);
		expect(shouldStartAutoplayCountdown(false, true, true)).toBe(false);
		expect(shouldStartAutoplayCountdown(true, false, true)).toBe(false);
		expect(shouldStartAutoplayCountdown(true, true, false)).toBe(false);
	});

	it('construye la URL de recarga conservando audio, calidad y tiempo', () => {
		expect(
			playbackReloadURL(7, { audio: 1, quality: '720', subtitle: 'e1', time: 42.9 })
		).toBe('/items/7/play?audio=1&quality=720&subtitle=e1&t=42');
		expect(playbackReloadURL(7, { subtitle: null, time: 0 })).toBe(
			'/items/7/play?subtitle=&t=0'
		);
		expect(playbackReloadURL(7, { subtitle: 'e0', time: 3.8 })).toBe(
			'/items/7/play?subtitle=e0&t=3'
		);
	});
});
