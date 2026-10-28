import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { Session } from '$lib/api';
import { formatDateTime } from '$lib/media';
import SessionList from './SessionList.svelte';

function session(overrides: Partial<Session> = {}): Session {
	return {
		id: 7,
		device_name: 'Portátil',
		client: 'web',
		created_at: '2026-10-27T09:00:00Z',
		last_seen_at: '2026-10-27T10:00:00Z',
		expires_at: '2026-10-28T10:00:00Z',
		current: false,
		...overrides
	};
}

function body(props: Record<string, unknown>): string {
	return render(SessionList, { props: props as never }).body;
}

describe('lista de sesiones', () => {
	it('avisa cuando no hay sesiones', () => {
		const html = body({ sessions: [] });
		expect(html).toContain('No hay sesiones abiertas.');
	});

	it('distingue una lista que no se pudo cargar', () => {
		const html = body({ sessions: null });
		expect(html).toContain('No se pudieron cargar las sesiones.');
		expect(html).not.toContain('No hay sesiones abiertas.');
	});

	it('muestra la sesión actual con sus datos y el cierre específico', () => {
		const current = session({ current: true });
		const html = body({
			sessions: [current],
			formAction: '?/closeSession',
			hiddenFields: (item: Session) => ({ id: item.id, _current: item.current ? '1' : '0' }),
			currentLabel: 'Este dispositivo'
		});

		expect(html).toContain('Portátil');
		expect(html).toContain('Navegador');
		expect(html).toContain('Este dispositivo');
		expect(html).toContain(`Última actividad: ${formatDateTime(current.last_seen_at)}`);
		expect(html).toContain(`Caduca: ${formatDateTime(current.expires_at)}`);
		expect(html).toContain('Cerrar esta sesión');
		expect(html).toContain('action="?/closeSession"');
		expect(html).toContain('name="id"');
		expect(html).toContain('value="7"');
	});

	it('usa un nombre alternativo y el tipo de cliente', () => {
		const html = body({
			sessions: [
				session({ id: 8, device_name: '', client: 'androidtv' }),
				session({ id: 9, device_name: 'Móvil', client: 'android' })
			],
			hiddenFields: { user_id: 4 }
		});
		expect(html).toContain('Dispositivo sin nombre');
		expect(html).toContain('Android TV');
		expect(html).toContain('Android');
		expect(html).toContain('>Cerrar<');
		expect(html).not.toContain('Cerrar esta sesión');
		expect(html).toContain('name="user_id"');
		expect(html).toContain('value="4"');
	});
});
