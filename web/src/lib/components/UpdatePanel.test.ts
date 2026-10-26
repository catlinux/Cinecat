import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { UpdateInfo, UpdateStatus } from '$lib/api';
import UpdatePanel from './UpdatePanel.svelte';

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
		latest: 'v0.16.0',
		available: true,
		checked_at: '2026-10-27T10:00:00Z',
		notes: '',
		apply_supported: true,
		check_enabled: true,
		status: status(),
		...overrides
	};
}

function body(props: Record<string, unknown>): string {
	return render(UpdatePanel, { props: props as never }).body;
}

describe('panel de actualizaciones', () => {
	it('muestra que está al día y la fecha de comprobación', () => {
		const html = body({
			info: info({ available: false, latest: 'v0.15.0', current: '0.15.0' })
		});
		expect(html).toContain('Estás al día (0.15.0)');
		expect(html).toContain('Comprobado:');
		expect(html).toContain('Buscar actualizaciones');
	});

	it('muestra la versión nueva y escapa las notas', () => {
		const html = body({
			info: info({ notes: '<script>alert("no")</script>\n- Cambio' })
		});
		expect(html).toContain('Hay una versión nueva: v0.16.0');
		expect(html).toContain('Notas de la versión');
		expect(html).toContain('&lt;script>alert("no")&lt;/script>');
		expect(html).not.toContain('<script>alert("no")</script>');
	});

	it('pide confirmación antes de aplicar', () => {
		const html = body({ info: info() });
		expect(html).toContain('Se hará una copia de la base de datos');
		expect(html).toContain('¿Continuar?');
		expect(html).toContain('action="?/applyUpdate"');
		expect(html).toContain('name="tag"');
		expect(html).toContain('value="v0.16.0"');
	});

	it('explica cómo instalar el actualizador si no está disponible', () => {
		const html = body({ info: info({ apply_supported: false }) });
		expect(html).toContain('La actualización automática no está instalada en este servidor');
		expect(html).toContain('./scripts/instalar-actualizador.sh');
		expect(html).toContain('./scripts/actualizar.sh');
		expect(html).not.toContain('action="?/applyUpdate"');
	});

	it('avisa de la búsqueda automática desactivada sin ocultar el botón manual', () => {
		const html = body({ info: info({ check_enabled: false }) });
		expect(html).toContain('La búsqueda automática está desactivada; puedes comprobar a mano');
		expect(html).toContain('Buscar actualizaciones');
	});

	it('muestra la fase y el progreso indeterminado', () => {
		const html = body({
			info: info({
				available: false,
				status: status({
					phase: 'Compilando',
					started_at: '2026-10-27T10:00:00Z'
				})
			})
		});
		expect(html).toContain('Compilando la versión nueva');
		expect(html).toContain('Tiempo transcurrido:');
		expect(html).toContain('role="progressbar"');
		expect(html).toContain('aria-live="polite"');
	});

	it('entra en seguimiento después de solicitar la actualización', () => {
		const html = body({
			info: info({
				status: status({ phase: 'Terminado', tag: 'v0.15.0', result: 'ok' })
			}),
			form: {
				updateAction: true,
				updateInfo: info({
					status: status({ phase: 'Terminado', tag: 'v0.15.0', result: 'ok' })
				}),
				appliedTag: 'v0.16.0'
			}
		});
		expect(html).toContain('Actualización en curso');
		expect(html).toContain('Tiempo transcurrido:');
		expect(html).not.toContain('Actualizado a v0.15.0');
		expect(html).not.toContain('action="?/applyUpdate"');
	});

	it('trata los errores de sondeo como reinicio', () => {
		const html = body({
			info: info({
				status: status({ phase: 'Reiniciando', started_at: '2026-10-27T10:00:00Z' })
			}),
			initialPollError: true
		});
		expect(html).toContain('Reiniciando el servidor…');
		expect(html).toContain('es normal que la web no responda');
	});

	it('anuncia la actualización terminada', () => {
		const html = body({
			info: info({
				current: '0.16.0',
				status: status({ phase: 'Terminado', tag: 'v0.16.0', result: 'ok' })
			})
		});
		expect(html).toContain('Actualizado a v0.16.0');
		expect(html).toContain('Recargar la página');
	});

	it('muestra el fallo y el motivo', () => {
		const html = body({
			info: info({
				available: false,
				status: status({ result: 'failed', message: 'No quedaba espacio.' })
			})
		});
		expect(html).toContain('La actualización falló. Revisa el servicio en el servidor.');
		expect(html).toContain('No quedaba espacio.');
	});

	it('muestra la reversión y su motivo', () => {
		const html = body({
			info: info({
				available: false,
				current: '0.15.0',
				status: status({ result: 'rolled_back', message: 'La versión nueva no arrancó.' })
			})
		});
		expect(html).toContain('La actualización falló y se volvió a la versión anterior');
		expect(html).toContain('La versión nueva no arrancó.');
	});

	it('muestra el enlace de actualización sin JavaScript', () => {
		const html = body({
			info: info({
				status: status({ phase: 'Validando', started_at: '2026-10-27T10:00:00Z' })
			})
		});
		expect(html).toContain('<noscript>');
		expect(html).toContain('Actualizar estado');
	});
});
