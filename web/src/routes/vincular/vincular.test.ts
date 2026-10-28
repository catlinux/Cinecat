import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import { formatDateTime } from '$lib/media';
import VincularPage from './+page.svelte';
import { actions, load } from './+page.server';

const previewAction = actions.preview!;
const approveAction = actions.approve!;

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function loadEvent(fetchImpl: typeof fetch, url: string, token: string | null = 'token-123') {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(url),
		params: {},
		cookies: { get: vi.fn(), set: vi.fn(), delete: vi.fn() },
		route: { id: '/vincular' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	action: string,
	fields: Record<string, string>,
	token: string | null = 'token-123',
	url = 'http://localhost/vincular?codigo=ABCD-EFGH'
) {
	return {
		request: new Request(`http://localhost/vincular?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		cookies: { get: vi.fn(), set: vi.fn(), delete: vi.fn() },
		url: new URL(url),
		params: {},
		route: { id: '/vincular' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

const preview = {
	device_name: 'Salón',
	client: 'androidtv',
	expires_at: '2026-10-28T22:30:00Z'
};

describe('carga de la vinculación', () => {
	it('manda al login con next cuando no hay sesión', async () => {
		const fetchMock = vi.fn();
		await expect(
			load(
				loadEvent(fetchMock, 'http://localhost/vincular?codigo=ABCD-EFGH', null) as never
			)
		).rejects.toMatchObject({
			status: 303,
			location: '/login?next=%2Fvincular%3Fcodigo%3DABCD-EFGH'
		});
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('prellena el código de la query con sesión', async () => {
		const result = (await load(
			loadEvent(vi.fn(), 'http://localhost/vincular?codigo=ABCD-EFGH') as never
		)) as { initialCode: string };
		expect(result.initialCode).toBe('ABCD-EFGH');
	});
});

describe('acciones de vinculación', () => {
	it('hace la vista previa sin aprobar', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(preview));
		const result = (await previewAction(
			actionEvent(fetchMock, 'preview', { code: 'ABCD-EFGH' }) as never
		)) as {
			step: string;
			code: string;
			deviceName: string;
			client: string;
			expiresAt: string;
		};

		expect(result.step).toBe('preview');
		expect(result.deviceName).toBe('Salón');
		expect(result.client).toBe('androidtv');
		expect(result.expiresAt).toBe(preview.expires_at);
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/auth/pair/approve');
		expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer token-123');
		expect(JSON.parse(String(fetchMock.mock.calls[0][1].body))).toEqual({
			code: 'ABCD-EFGH',
			confirm: false
		});
	});

	it('aprueba la vinculación', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ device_name: 'Salón', client: 'androidtv' })
		);
		const result = (await approveAction(
			actionEvent(fetchMock, 'approve', { code: 'ABCD-EFGH' }) as never
		)) as { step: string; deviceName: string };
		expect(result.step).toBe('approved');
		expect(result.deviceName).toBe('Salón');
		expect(JSON.parse(String(fetchMock.mock.calls[0][1].body))).toEqual({
			code: 'ABCD-EFGH',
			confirm: true
		});
	});

	it('traduce los errores del servidor', async () => {
		const cases = [
			[400, 'Ese código no tiene el formato correcto.'],
			[404, 'El código no existe o ha caducado. Pide uno nuevo en el dispositivo.'],
			[429, 'Demasiados intentos. Espera unos minutos.']
		] as const;
		for (const [status, message] of cases) {
			const fetchMock = vi.fn().mockResolvedValue(
				jsonResponse({ error: 'error' }, status)
			);
			const result = (await previewAction(
				actionEvent(fetchMock, 'preview', { code: 'ABCD-EFGH' }) as never
			)) as { status: number; data: { message: string } };
			expect(result.status).toBe(status);
			expect(result.data.message).toBe(message);
		}
	});

	it('redirige al login si la sesión caducó durante la acción', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));
		await expect(
			previewAction(actionEvent(fetchMock, 'preview', { code: 'ABCD-EFGH' }) as never)
		).rejects.toMatchObject({
			status: 303,
			location: '/login?next=%2Fvincular%3Fcodigo%3DABCD-EFGH'
		});
	});

	it('no llama a la API sin sesión', async () => {
		const fetchMock = vi.fn();
		await expect(
			approveAction(actionEvent(fetchMock, 'approve', { code: 'ABCD-EFGH' }, null) as never)
		).rejects.toMatchObject({
			status: 303,
			location: '/login?next=%2Fvincular%3Fcodigo%3DABCD-EFGH'
		});
		expect(fetchMock).not.toHaveBeenCalled();
	});
});

describe('página de vinculación', () => {
	it('prellena el código y ofrece el formulario', () => {
		const html = render(VincularPage, {
			props: {
				data: { ui: { server_name: 'Cinecat' }, initialCode: 'ABCD-EFGH' } as never,
				form: null
			}
		}).body;

		expect(html).toContain('Vincular dispositivo');
		expect(html).toContain('value="ABCD-EFGH"');
		expect(html).toContain('autocapitalize="characters"');
		expect(html).toContain('autocomplete="off"');
		expect(html).toContain('inputmode="text"');
		expect(html).toContain('maxlength="9"');
		expect(html).toContain('placeholder="XXXX-XXXX"');
		expect(html).toContain('action="?/preview"');
		expect(html).toContain('Continuar');
	});

	it('muestra la vista previa con el aviso y los datos del dispositivo', () => {
		const html = render(VincularPage, {
			props: {
				data: { ui: { server_name: 'Cinecat' }, initialCode: '' } as never,
				form: {
					step: 'preview',
					code: 'ABCD-EFGH',
					deviceName: 'Salón',
					client: 'androidtv',
					expiresAt: preview.expires_at
				}
			}
		}).body;

		expect(html).toContain('Salón');
		expect(html).toContain('Android TV');
		expect(html).toContain(formatDateTime(preview.expires_at));
		expect(html).toContain(
			'Aprueba solo un código que aparezca ahora mismo en la pantalla de tu dispositivo. Si alguien te ha pasado este código, no lo apruebes: daría acceso a tu cuenta.'
		);
		expect(html).toContain('action="?/approve"');
		expect(html).toContain('name="code"');
		expect(html).toContain('value="ABCD-EFGH"');
		expect(html).toContain('Vincular');
		expect(html).toContain('Cancelar');
	});

	it('usa el nombre alternativo cuando el dispositivo no tiene nombre', () => {
		const html = render(VincularPage, {
			props: {
				data: { ui: { server_name: 'Cinecat' }, initialCode: '' } as never,
				form: {
					step: 'preview',
					code: 'ABCD-EFGH',
					deviceName: '',
					client: 'android',
					expiresAt: preview.expires_at
				}
			}
		}).body;
		expect(html).toContain('Dispositivo sin nombre');
		expect(html).toContain('Android');
	});

	it('anuncia la vinculación terminada y enlaza a Tu cuenta', () => {
		const html = render(VincularPage, {
			props: {
				data: { ui: { server_name: 'Cinecat' }, initialCode: '' } as never,
				form: { step: 'approved', deviceName: 'Salón', client: 'androidtv' }
			}
		}).body;
		expect(html).toContain('Listo. Ya puedes usar Cinecat en «Salón».');
		expect(html).toContain('cuando quieras en');
		expect(html).toContain('href="/account"');
		expect(html).toContain('role="status"');
	});

	it('muestra el error con foco accesible', () => {
		const html = render(VincularPage, {
			props: {
				data: { ui: { server_name: 'Cinecat' }, initialCode: '' } as never,
				form: {
					message: 'El código no existe o ha caducado. Pide uno nuevo en el dispositivo.',
					code: 'ABCD-EFGH'
				}
			}
		}).body;
		expect(html).toContain('role="alert"');
		expect(html).toContain('tabindex="-1"');
		expect(html).toContain('El código no existe o ha caducado');
	});
});
