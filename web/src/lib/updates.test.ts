import { describe, expect, it, vi } from 'vitest';
import {
	buildUpdateCheck,
	compareVersions,
	fetchLatestVersion,
	formatVersion,
	isNewerVersion,
	parseVersion
} from './updates';

describe('versiones SemVer', () => {
	it('compara versiones iguales y mayores', () => {
		expect(compareVersions('1.2.3', '1.2.3')).toBe(0);
		expect(compareVersions('2.0.0', '1.9.9')).toBe(1);
		expect(compareVersions('0.11.0', '0.11.1')).toBe(-1);
		expect(isNewerVersion('1.0.0', '0.9.9')).toBe(true);
		expect(isNewerVersion('0.9.9', '1.0.0')).toBe(false);
	});

	it('ordena las etiquetas de prereléase', () => {
		expect(compareVersions('1.0.0-alpha', '1.0.0')).toBe(-1);
		expect(compareVersions('1.0.0', '1.0.0-beta')).toBe(1);
		expect(compareVersions('1.0.0-alpha.1', '1.0.0-alpha.2')).toBe(-1);
		expect(compareVersions('1.0.0-alpha.2', '1.0.0-alpha.beta')).toBe(-1);
		expect(compareVersions('1.0.0-alpha', '1.0.0-beta')).toBe(-1);
	});

	it('admite espacios y saltos de línea exteriores', () => {
		expect(parseVersion('  1.2.3\n')).toEqual({
			major: 1,
			minor: 2,
			patch: 3,
			prerelease: []
		});
		expect(parseVersion('v1.2.3')).not.toBeNull();
		expect(formatVersion(parseVersion('1.2.3-alpha.1')!)).toBe('1.2.3-alpha.1');
	});

	it('rechaza formatos inválidos', () => {
		for (const value of ['', '1.2', '1.2.3.4', '1.2.x', '01.2.3', '1.0.0-01', 'dos']) {
			expect(parseVersion(value)).toBeNull();
			expect(compareVersions(value, '1.0.0')).toBeNull();
		}
	});
});

describe('resultado de actualización', () => {
	it('avisa cuando hay una versión nueva', () => {
		const result = buildUpdateCheck('2.0.0\n', '0.11.0');
		expect(result.upToDate).toBe(false);
		expect(result.message).toContain('2.0.0');
	});

	it('indica que está al día', () => {
		const result = buildUpdateCheck('0.11.0', '0.11.0');
		expect(result.upToDate).toBe(true);
		expect(result.message).toContain('0.11.0');
	});
});

describe('consulta de la versión publicada', () => {
	it('lee y valida el archivo remoto', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(' 1.2.3\n', { status: 200 })
		);
		await expect(fetchLatestVersion(fetchMock)).resolves.toBe('1.2.3');
	});

	it('propaga los errores de red, estado y formato', async () => {
		const network = vi.fn().mockRejectedValue(new Error('sin conexión'));
		await expect(fetchLatestVersion(network)).rejects.toThrow('sin conexión');

		const status = vi.fn().mockResolvedValue(new Response('', { status: 503 }));
		await expect(fetchLatestVersion(status)).rejects.toThrow('503');

		const invalid = vi.fn().mockResolvedValue(new Response('versión rota', { status: 200 }));
		await expect(fetchLatestVersion(invalid)).rejects.toThrow('formato válido');
	});
});
