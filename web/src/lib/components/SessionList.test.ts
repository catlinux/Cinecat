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

	it('muestra los ajustes del perfil solo cuando se activan', () => {
		const html = body({
			sessions: [
				session({
					profile: {
						max_quality: '720',
						audio_language: 'spa',
						subtitle_language: 'off'
					}
				})
			],
			profileAction: '?/saveDeviceProfile'
		});

		expect(html).toContain('Ajustes');
		expect(html).toContain('Calidad máxima');
		expect(html).toContain('Idioma de audio preferido');
		expect(html).toContain('Subtítulos por defecto');
		expect(html).toContain('action="?/saveDeviceProfile"');
		expect(html).toMatch(/value="720"[^>]*selected/);
		expect(html).toMatch(/value="spa"[^>]*selected/);
		expect(html).toContain('>spa<');
		expect(html).toMatch(/value="off"[^>]*selected/);
		expect(html).toContain(
			'Se aplican al reproducir en este dispositivo cuando no hayas elegido otra cosa en la película.'
		);
	});

	it('usa un perfil vacío si el servidor antiguo no lo envía', () => {
		const html = body({
			sessions: [session()],
			profileAction: '?/saveDeviceProfile'
		});
		expect(html).toContain('Calidad máxima');
		expect(html).toMatch(/value=""[^>]*selected/);
	});

	it('no muestra ajustes fuera de Mi cuenta', () => {
		const html = body({ sessions: [session()] });
		expect(html).not.toContain('Calidad máxima');
		expect(html).not.toContain('Ajustes');
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
