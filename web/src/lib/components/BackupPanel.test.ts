import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { Backup, BackupsResponse } from '$lib/api';
import { formatBytes, formatDateTime } from '$lib/media';
import BackupPanel from './BackupPanel.svelte';

function backup(overrides: Partial<Backup> = {}): Backup {
	return {
		name: 'cinecat-20261027-100000.db',
		size_bytes: 734003200,
		created_at: '2026-10-27T10:00:00Z',
		...overrides
	};
}

function info(overrides: Partial<BackupsResponse> = {}): BackupsResponse {
	return {
		enabled: true,
		retention_days: 7,
		backups: [],
		last_error: '',
		...overrides
	};
}

function body(props: Record<string, unknown>): string {
	return render(BackupPanel, { props: props as never }).body;
}

describe('panel de copias de seguridad', () => {
	it('muestra el estado y que no hay copias', () => {
		const html = body({ info: info() });
		expect(html).toContain('Copias automáticas');
		expect(html).toContain('Activadas');
		expect(html).toContain('7 días');
		expect(html).toContain('se cambia en los ajustes de arriba');
		expect(html).toContain('No hay copias todavía.');
		expect(html).toContain('Hacer copia ahora');
		expect(html).toContain('action="?/backup"');
		expect(html).toContain('aria-live="polite"');
	});

	it('indica cuando las copias automáticas están desactivadas', () => {
		const html = body({ info: info({ enabled: false }) });
		expect(html).toContain('Desactivadas');
	});

	it('lista las copias ordenadas, con fecha y tamaño legible', () => {
		const older = backup({
			name: 'cinecat-20261026-090000.db',
			size_bytes: 1500000000,
			created_at: '2026-10-26T09:00:00Z'
		});
		const newer = backup();
		const html = body({ info: info({ backups: [older, newer] }) });

		expect(html).toContain('Última copia:');
		expect(html).toContain(formatDateTime(newer.created_at));
		expect(html).toContain(newer.name);
		expect(html).toContain(older.name);
		expect(html).toContain(formatBytes(newer.size_bytes));
		expect(html).toContain(formatBytes(older.size_bytes));
		expect(html.indexOf(newer.name)).toBeLessThan(html.indexOf(older.name));
	});

	it('muestra el último error del servidor', () => {
		const html = body({ info: info({ last_error: 'No se pudo escribir en el directorio.' }) });
		expect(html).toContain('Último error: No se pudo escribir en el directorio.');
	});

	it('avisa si no se pudo cargar la lista sin ocultar el botón', () => {
		const html = body({ info: null, loadError: true });
		expect(html).toContain('No se pudieron cargar las copias de seguridad');
		expect(html).toContain('Hacer copia ahora');
		expect(html).not.toContain('Copias automáticas');
	});

	it('muestra el mensaje de una copia en curso', () => {
		const html = body({
			info: info(),
			form: {
				backupAction: true,
				success: false,
				message: 'Ya hay una copia en curso'
			}
		});
		expect(html).toContain('Ya hay una copia en curso');
		expect(html).toContain('role="alert"');
	});

	it('muestra el éxito de la copia', () => {
		const html = body({
			info: info(),
			form: {
				backupAction: true,
				success: true,
				message: 'Copia de seguridad creada.'
			}
		});
		expect(html).toContain('Copia de seguridad creada.');
		expect(html).toContain('role="status"');
	});
});
