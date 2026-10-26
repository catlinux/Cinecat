import { describe, expect, it } from 'vitest';
import {
	PROGRESS_INTERVAL_MS,
	decideProgress,
	episodeProgressSubtitle,
	formatProgressTime,
	progressPercent,
	resolveStartAt,
	resumeTarget
} from './progress';

describe('decisión de envío del progreso', () => {
	it('envía la primera muestra', () => {
		const decision = decideProgress('timeupdate', { position: 90, duration: 600 }, null, 1000);
		expect(decision).toEqual({ send: true, position: 90, duration: 600 });
	});

	it('limita los timeupdate al intervalo de 15 segundos', () => {
		const before = decideProgress(
			'timeupdate',
			{ position: 100, duration: 600 },
			1000,
			1000 + PROGRESS_INTERVAL_MS - 1
		);
		expect(before.send).toBe(false);

		const after = decideProgress(
			'timeupdate',
			{ position: 100, duration: 600 },
			1000,
			1000 + PROGRESS_INTERVAL_MS
		);
		expect(after.send).toBe(true);
	});

	it('envía al pausar y al abandonar la página sin esperar el intervalo', () => {
		expect(decideProgress('pause', { position: 100, duration: 600 }, 999, 1000).send).toBe(true);
		expect(decideProgress('pagehide', { position: 100, duration: 600 }, 999, 1000).send).toBe(
			true
		);
	});

	it('al terminar envía la duración completa', () => {
		const decision = decideProgress('ended', { position: 590, duration: 600 }, 999, 1000);
		expect(decision.send).toBe(true);
		expect(decision.position).toBe(600);
		expect(decision.duration).toBe(600);
	});

	it('ignora duraciones no finitas o cero y posiciones negativas', () => {
		expect(decideProgress('timeupdate', { position: 10, duration: Number.NaN }, null, 0).send).toBe(
			false
		);
		expect(decideProgress('timeupdate', { position: 10, duration: 0 }, null, 0).send).toBe(false);
		expect(decideProgress('timeupdate', { position: -1, duration: 600 }, null, 0).send).toBe(
			false
		);
	});
});

describe('cálculo y formato del progreso', () => {
	it('calcula el porcentaje entre 0 y 100', () => {
		expect(progressPercent(0, 600)).toBe(0);
		expect(progressPercent(300, 600)).toBe(50);
		expect(progressPercent(700, 600)).toBe(100);
		expect(progressPercent(10, 0)).toBe(0);
	});

	it('formatea la posición en horas y minutos', () => {
		expect(formatProgressTime(4320)).toBe('1 h 12 min');
		expect(formatProgressTime(90)).toBe('1 min');
		expect(formatProgressTime(45)).toBe('45 s');
		expect(formatProgressTime(null)).toBe('');
	});

	it('resuelve el punto de inicio con o sin ?t=', () => {
		expect(resolveStartAt({ position_seconds: 120 }, null)).toBe(120);
		expect(resolveStartAt({ position_seconds: 120 }, '30')).toBe(30);
		expect(resolveStartAt({ position_seconds: 120 }, '0')).toBe(0);
		expect(resolveStartAt({ position_seconds: 120 }, 'no-es-un-numero')).toBe(120);
		expect(resolveStartAt(null, null)).toBe(0);
	});

	it('describe el episodio en la fila de continuar viendo', () => {
		expect(episodeProgressSubtitle({ kind: 'episode', season: 2, episode: 5 })).toBe(
			'Serie · T2 E5'
		);
		expect(episodeProgressSubtitle({ kind: 'movie', season: null, episode: null })).toBe('');
	});

	it('aplica el margen de 15 segundos al reanudar', () => {
		expect(resumeTarget(50, 100, false)).toBe(50);
		expect(resumeTarget(86, 100, false)).toBeNull();
		expect(resumeTarget(50, 100, true)).toBeNull();
	});
});
