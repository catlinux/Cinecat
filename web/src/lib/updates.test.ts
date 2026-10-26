import { describe, expect, it } from 'vitest';
import type { UpdateInfo, UpdateStatus } from './api';
import {
	compareVersions,
	elapsedSeconds,
	formatElapsed,
	isUpdateInProgress,
	parseVersion,
	phaseLabel,
	pollDelay,
	trackingExpired,
	updateOutcome,
	updatedTag,
	versionMatches
} from './updates';

function status(overrides: Partial<UpdateStatus> = {}): UpdateStatus {
	return {
		phase: '',
		tag: '',
		message: '',
		started_at: '',
		finished_at: '',
		result: '',
		...overrides
	};
}

function info(overrides: Partial<UpdateInfo> = {}): UpdateInfo {
	return {
		current: '0.15.0',
		latest: '',
		available: false,
		checked_at: '',
		notes: '',
		apply_supported: true,
		check_enabled: true,
		status: status(),
		...overrides
	};
}

describe('versiones SemVer', () => {
	it('compara versiones iguales y mayores', () => {
		expect(compareVersions('1.2.3', '1.2.3')).toBe(0);
		expect(compareVersions('2.0.0', '1.9.9')).toBe(1);
		expect(compareVersions('0.11.0', '0.11.1')).toBe(-1);
		expect(compareVersions('v0.16.0', '0.16.0')).toBe(0);
	});

	it('ordena las etiquetas de prerelación', () => {
		expect(compareVersions('1.0.0-alpha', '1.0.0')).toBe(-1);
		expect(compareVersions('1.0.0', '1.0.0-beta')).toBe(1);
		expect(compareVersions('1.0.0-alpha.1', '1.0.0-alpha.2')).toBe(-1);
		expect(compareVersions('1.0.0-alpha.2', '1.0.0-alpha.beta')).toBe(-1);
		expect(compareVersions('1.0.0-alpha', '1.0.0-beta')).toBe(-1);
	});

	it('admite espacios y rechaza formatos inválidos', () => {
		expect(parseVersion('  1.2.3\n')).toEqual({
			major: 1,
			minor: 2,
			patch: 3,
			prerelease: []
		});
		for (const value of ['', '1.2', '1.2.3.4', '1.2.x', '01.2.3', '1.0.0-01', 'dos']) {
			expect(parseVersion(value)).toBeNull();
			expect(compareVersions(value, '1.0.0')).toBeNull();
		}
	});
});

describe('interpretación del estado', () => {
	it('detecta una actualización en curso solo sin resultado', () => {
		expect(isUpdateInProgress(status({ phase: 'Compilando' }))).toBe(true);
		expect(isUpdateInProgress(status({ phase: 'Compilando', result: 'ok' }))).toBe(false);
		expect(isUpdateInProgress(status({ phase: '  ' }))).toBe(false);
		expect(isUpdateInProgress(null)).toBe(false);
	});

	it('traduce las fases conocidas y conserva las desconocidas', () => {
		expect(phaseLabel('Compilando')).toBe('Compilando la versión nueva');
		expect(phaseLabel('Copia de la base de datos')).toBe(
			'Guardando una copia de la base de datos'
		);
		expect(phaseLabel('Otra fase')).toBe('Otra fase');
		expect(phaseLabel('')).toBe('Actualización en curso');
	});

	it('compara la versión en ejecución con la etiqueta', () => {
		expect(versionMatches('0.16.0', 'v0.16.0')).toBe(true);
		expect(versionMatches('0.16.0', 'v0.17.0')).toBe(false);
		expect(versionMatches('desconocida', 'v0.16.0')).toBe(false);
	});

	it('da por terminada la actualización solo si la versión coincide', () => {
		const done = info({
			current: '0.16.0',
			status: status({ phase: 'Terminado', tag: 'v0.16.0', result: 'ok' })
		});
		expect(updatedTag(done)).toBe('v0.16.0');
		expect(updateOutcome(done)).toBe('ok');
		expect(updatedTag(info({ current: '0.15.0', status: done.status }))).toBeNull();
		expect(updatedTag(done, 'v0.16.0')).toBe('v0.16.0');
		expect(updatedTag(done, 'v0.17.0')).toBeNull();
		expect(updateOutcome(done, 'v0.17.0')).toBeNull();
	});

	it('distingue fallo y reversión', () => {
		expect(
			updateOutcome(info({ status: status({ result: 'failed', message: 'sin espacio' }) }))
		).toBe('failed');
		expect(
			updateOutcome(info({ status: status({ result: 'rolled_back', message: 'salud' }) }))
		).toBe('rolled_back');
		expect(
			updateOutcome(
				info({
					status: status({ result: 'failed', tag: 'v0.15.0', message: 'viejo' })
				}),
				'v0.16.0'
			)
		).toBeNull();
		expect(updateOutcome(info())).toBeNull();
	});
});

describe('tiempos y sondeo', () => {
	it('calcula el tiempo transcurrido', () => {
		const start = '2026-10-27T10:00:00Z';
		expect(elapsedSeconds(start, Date.parse('2026-10-27T10:01:05Z'))).toBe(65);
		expect(elapsedSeconds('', Date.now())).toBe(0);
		expect(elapsedSeconds('no es una fecha', Date.now())).toBe(0);
		expect(formatElapsed(65)).toBe('1:05');
		expect(formatElapsed(3725)).toBe('1:02:05');
	});

	it('aplica backoff hasta 10 segundos', () => {
		expect(pollDelay(0)).toBe(3000);
		expect(pollDelay(1)).toBe(6000);
		expect(pollDelay(2)).toBe(10000);
		expect(pollDelay(20)).toBe(10000);
		expect(pollDelay(-1)).toBe(3000);
	});

	it('deja de seguir a los 30 minutos', () => {
		const start = '2026-10-27T10:00:00Z';
		expect(trackingExpired(start, Date.parse('2026-10-27T10:29:59Z'))).toBe(false);
		expect(trackingExpired(start, Date.parse('2026-10-27T10:30:00Z'))).toBe(true);
		expect(trackingExpired('', Date.now())).toBe(false);
	});
});
