import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type {
	ScanStatus,
	ServerVersion,
	Setting,
	SettingsResponse,
	SettingsSection
} from '$lib/api';
import AdminNav from '$lib/components/AdminNav.svelte';
import SettingsForm from '$lib/components/SettingsForm.svelte';
import { webVersion } from '$lib/version';
import AdminLayout from './+layout.svelte';
import { load as loadAdminLayout } from './+layout.server';
import AdminSummary from './+page.svelte';
import { load as loadSummary } from './+page.server';
import SettingsPage from './settings/[section]/+page.svelte';
import { actions as settingsActions, load as loadSettings } from './settings/[section]/+page.server';

const save = settingsActions.save!;
const reset = settingsActions.reset!;
const checkUpdates = settingsActions.checkUpdates!;

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function makeSetting(overrides: Partial<Setting> & Pick<Setting, 'key' | 'type'>): Setting {
	return {
		label: overrides.key,
		description: 'Descripción del ajuste.',
		options: [],
		min: 0,
		max: 0,
		default: null,
		value: null,
		modified: false,
		...overrides
	};
}

function scanSection(): SettingsSection {
	return {
		key: 'scan',
		label: 'Escaneo',
		settings: [
			makeSetting({
				key: 'scan.interval',
				type: 'duration',
				label: 'Intervalo de escaneo',
				default: '6h',
				value: '6h',
				min: 60
			}),
			makeSetting({
				key: 'scan.on_startup',
				type: 'bool',
				label: 'Escanear al arrancar',
				default: true,
				value: true
			})
		]
	};
}

function metadataSection(): SettingsSection {
	return {
		key: 'metadata',
		label: 'Metadatos',
		settings: [
			makeSetting({
				key: 'metadata.language',
				type: 'enum',
				label: 'Idioma de los metadatos',
				default: 'es-ES',
				value: 'es-ES',
				options: [
					{ value: 'es-ES', label: 'Español (España)' },
					{ value: 'en-US', label: 'Inglés' }
				]
			}),
			makeSetting({
				key: 'metadata.match_threshold',
				type: 'float',
				label: 'Umbral de identificación',
				default: 0.75,
				value: 0.75,
				min: 0.6,
				max: 0.95
			})
		]
	};
}

function uiSection(): SettingsSection {
	return {
		key: 'ui',
		label: 'Interfaz',
		settings: [
			makeSetting({
				key: 'ui.server_name',
				type: 'string',
				label: 'Nombre del servidor',
				default: 'Cinecat',
				value: 'Cinecat',
				min: 1,
				max: 40
			}),
			makeSetting({
				key: 'ui.grid_columns',
				type: 'int',
				label: 'Columnas de los listados',
				default: 6,
				value: 6,
				min: 4,
				max: 8
			}),
			makeSetting({
				key: 'ui.show_series',
				type: 'bool',
				label: 'Mostrar series',
				default: true,
				value: true
			})
		]
	};
}

function playbackSection(): SettingsSection {
	return {
		key: 'playback',
		label: 'Reproducción',
		settings: [
			makeSetting({
				key: 'playback.max_height',
				type: 'enum',
				label: 'Altura máxima',
				default: 1080,
				value: 1080,
				options: [
					{ value: '480', label: '480p' },
					{ value: '720', label: '720p' },
					{ value: '1080', label: '1080p' }
				],
				min: 480,
				max: 1080
			}),
			makeSetting({
				key: 'playback.crf',
				type: 'int',
				label: 'Calidad (CRF)',
				default: 20,
				value: 20,
				min: 18,
				max: 28
			})
		]
	};
}

function systemSection(checkUpdatesEnabled = true): SettingsSection {
	return {
		key: 'system',
		label: 'Sistema',
		settings: [
			makeSetting({
				key: 'system.check_updates',
				type: 'bool',
				label: 'Buscar actualizaciones',
				default: false,
				value: checkUpdatesEnabled
			})
		]
	};
}

function settingsResponse(checkUpdatesEnabled = true): SettingsResponse {
	return {
		sections: [
			scanSection(),
			metadataSection(),
			uiSection(),
			playbackSection(),
			systemSection(checkUpdatesEnabled)
		]
	};
}

function section(response: SettingsResponse, key: string): SettingsSection {
	const found = response.sections.find((item) => item.key === key);
	if (!found) {
		throw new Error(`Falta la sección ${key}`);
	}
	return found;
}

function loadEvent(fetchImpl: typeof fetch, path: string, token: string | null = 'token-123') {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		params: {},
		cookies: {},
		route: { id: path },
		request: new Request(`http://localhost${path}`),
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function sectionLoadEvent(
	fetchImpl: typeof fetch,
	sectionKey: string,
	token: string | null = 'token-123'
) {
	return {
		...loadEvent(fetchImpl, `/admin/settings/${sectionKey}`, token),
		params: { section: sectionKey }
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	action: string,
	fields: Record<string, string>,
	sectionKey = 'scan',
	token: string | null = 'token-123'
) {
	return {
		request: new Request(`http://localhost/admin/settings/${sectionKey}?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(`http://localhost/admin/settings/${sectionKey}`),
		params: { section: sectionKey },
		cookies: {},
		route: { id: `/admin/settings/${sectionKey}` },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

const scanStatus: ScanStatus = {
	running: false,
	started_at: '2026-10-26T08:00:00Z',
	finished_at: '2026-10-26T08:05:00Z',
	libraries: 2,
	files: 12,
	upserted: 10,
	identified: 8,
	unidentified: 2,
	series_identified: 3,
	series_unidentified: 1,
	warnings: [],
	error: ''
};

const serverInfo: ServerVersion = {
	server: webVersion,
	commit: 'abc1234',
	built_at: '2026-10-26T09:00:00Z'
};

describe('layout de administración', () => {
	it('redirige sin sesión', async () => {
		await expect(
			loadAdminLayout(loadEvent(vi.fn(), '/admin', null) as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});

	it('rechaza a un usuario normal', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ username: 'pepe', role: 'user' })
		);
		await expect(loadAdminLayout(loadEvent(fetchMock, '/admin') as never)).rejects.toMatchObject({
			status: 403
		});
	});

	it('permite a un administrador y conserva la ruta', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ username: 'ana', role: 'admin' })
		);
		const result = (await loadAdminLayout(
			loadEvent(fetchMock, '/admin/settings/scan') as never
		)) as { adminUser: { username: string }; pathname: string };
		expect(result.adminUser.username).toBe('ana');
		expect(result.pathname).toBe('/admin/settings/scan');
	});

	it('marca la entrada activa y deja sitio para Usuarios', () => {
		const body = render(AdminNav, { props: { pathname: '/admin/settings/scan' } }).body;
		expect(body).toContain('aria-label="Administración"');
		expect(body).toContain('aria-current="page"');
		expect(body).toContain('Usuarios');
		expect(body).toContain('Pronto');
	});

	it('pinta la barra lateral y el contenido', () => {
		const children = createRawSnippet(() => ({
			render: () => '<p>Contenido del panel</p>'
		}));
		const body = render(AdminLayout, {
			props: {
				data: {
					adminUser: { username: 'ana', role: 'admin' },
					pathname: '/admin'
				} as never,
				children
			}
		}).body;
		expect(body).toContain('admin-shell');
		expect(body).toContain('Contenido del panel');
		expect(body).toContain('Resumen');
	});
});

describe('resumen de administración', () => {
	function summaryFetch(overrides: Partial<Record<string, () => Response>> = {}) {
		return vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			const key = `${url.pathname}${url.search}`;
			if (overrides[key]) {
				return overrides[key]();
			}
			switch (url.pathname) {
				case '/api/v1/items':
					if (url.searchParams.get('unmatched') === 'true') {
						return jsonResponse({ items: [], total: 2, limit: 1, offset: 0 });
					}
					return jsonResponse({ items: [], total: 10, limit: 1, offset: 0 });
				case '/api/v1/series':
					if (url.searchParams.get('unmatched') === 'true') {
						return jsonResponse({ series: [], total: 1, limit: 1, offset: 0 });
					}
					return jsonResponse({ series: [], total: 4, limit: 1, offset: 0 });
				case '/api/v1/collections':
					return jsonResponse({ collections: [], total: 3, limit: 1, offset: 0 });
				case '/api/v1/libraries/scan':
					return jsonResponse(scanStatus);
				case '/api/v1/system/version':
					return jsonResponse(serverInfo);
				default:
					throw new Error(`URL inesperada: ${url}`);
			}
		});
	}

	it('carga recuentos, escaneo y versiones', async () => {
		const result = (await loadSummary(loadEvent(summaryFetch(), '/admin') as never)) as {
			counts: {
				movies: number;
				unmatchedMovies: number;
				series: number;
				unmatchedSeries: number;
				collections: number;
			};
			scan: ScanStatus;
			serverInfo: ServerVersion;
		};
		expect(result.counts).toEqual({
			movies: 10,
			unmatchedMovies: 2,
			series: 4,
			unmatchedSeries: 1,
			collections: 3
		});
		expect(result.scan.files).toBe(12);
		expect(result.serverInfo.commit).toBe('abc1234');
	});

	it('no rompe el resumen si falla una petición', async () => {
		const fetchMock = summaryFetch({
			'/api/v1/collections?limit=1': () => jsonResponse({ error: 'fallo' }, 500)
		});
		const result = (await loadSummary(loadEvent(fetchMock, '/admin') as never)) as {
			counts: { collections: number | null; movies: number | null };
		};
		expect(result.counts.collections).toBeNull();
		expect(result.counts.movies).toBe(10);
	});

	it('redirige al login si el recuento devuelve 401', async () => {
		const fetchMock = summaryFetch({
			'/api/v1/items?limit=1&kind=movie': () => jsonResponse({ error: 'sesión' }, 401)
		});
		await expect(loadSummary(loadEvent(fetchMock, '/admin') as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('muestra guiones cuando faltan datos y no enlaza contadores sin identificar a cero', () => {
		const body = render(AdminSummary, {
			props: {
				data: {
					counts: {
						movies: null,
						unmatchedMovies: null,
						series: null,
						unmatchedSeries: null,
						collections: null
					},
					scan: null,
					serverInfo: null,
					webVersion
				} as never
			}
		}).body;

		expect(body).toContain('—');
		expect(body).toContain('Sin datos del último escaneo');
		expect(body).not.toContain('/movies?unmatched=1');
		expect(body).not.toContain('/series?unmatched=1');
	});

	it('muestra tarjetas, enlaces sin identificar y aviso de versiones', () => {
		const body = render(AdminSummary, {
			props: {
				data: {
					counts: {
						movies: 10,
						unmatchedMovies: 2,
						series: 4,
						unmatchedSeries: 1,
						collections: 3
					},
					scan: scanStatus,
					serverInfo: { ...serverInfo, server: '999.0.0' },
					webVersion
				} as never
			}
		}).body;

		expect(body).toContain('Resumen');
		expect(body).toContain('admin-summary-grid');
		expect((body.match(/admin-stat-card/g) ?? []).length).toBe(3);
		expect(body).toContain('10');
		expect(body).toContain('2 películas sin identificar');
		expect(body).toContain('/movies?unmatched=1');
		expect(body).toContain('1 series sin identificar');
		expect(body).toContain('/admin/libraries');
		expect(body).toContain('999.0.0');
		expect(body).toContain('versiones distintas');
		expect(body).toContain('12');
	});
});

describe('carga de ajustes', () => {
	it('carga cada sección del catálogo', async () => {
		const response = settingsResponse();
		for (const key of ['scan', 'metadata', 'ui', 'playback', 'system']) {
			const fetchMock = vi.fn().mockResolvedValue(jsonResponse(response));
			const result = (await loadSettings(
				sectionLoadEvent(fetchMock, key) as never
			)) as { section: SettingsSection };
			expect(result.section.key).toBe(key);
		}
	});

	it('devuelve 404 para una sección desconocida', async () => {
		const fetchMock = vi.fn();
		await expect(
			loadSettings(sectionLoadEvent(fetchMock, 'otra') as never)
		).rejects.toMatchObject({ status: 404 });
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('traduce 401, 403 y 502', async () => {
		const unauthorized = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));
		await expect(
			loadSettings(sectionLoadEvent(unauthorized, 'scan') as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });

		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'no admin' }, 403));
		await expect(
			loadSettings(sectionLoadEvent(forbidden, 'scan') as never)
		).rejects.toMatchObject({ status: 403 });

		const broken = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		await expect(
			loadSettings(sectionLoadEvent(broken, 'scan') as never)
		).rejects.toMatchObject({ status: 502 });
	});
});

describe('acciones de ajustes', () => {
	function sectionFetch(
		response: SettingsResponse,
		handler?: (url: URL, init?: RequestInit) => Response | undefined
	) {
		return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/settings' && (!init?.method || init.method === 'GET')) {
				return jsonResponse(response);
			}
			const handled = handler?.(url, init);
			if (handled) {
				return handled;
			}
			return jsonResponse(response);
		});
	}

	it('envía solo los campos cambiados', async () => {
		const response = settingsResponse();
		let patchBody: unknown = null;
		const fetchMock = sectionFetch(response, (url, init) => {
			if (url.pathname === '/api/v1/settings' && init?.method === 'PATCH') {
				patchBody = JSON.parse(String(init.body));
				return jsonResponse(response);
			}
			return undefined;
		});

		const result = (await save(
			actionEvent(fetchMock, 'save', {
				'scan.interval': '30m',
				'scan.on_startup': '1'
			}) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toBe('Ajustes guardados.');
		expect(patchBody).toEqual({ 'scan.interval': '30m' });
	});

	it('muestra los errores de validación del servidor junto a los valores escritos', async () => {
		const response = settingsResponse();
		const fetchMock = sectionFetch(response, (url, init) => {
			if (url.pathname === '/api/v1/settings' && init?.method === 'PATCH') {
				return jsonResponse(
					{ error: 'hay ajustes no válidos', fields: { 'scan.interval': 'Duración no válida' } },
					400
				);
			}
			return undefined;
		});

		const result = (await save(
			actionEvent(fetchMock, 'save', {
				'scan.interval': '30s',
				'scan.on_startup': '1'
			}) as never
		)) as { status: number; data: { fields: Record<string, string>; values: Record<string, string> } };
		expect(result.status).toBe(400);
		expect(result.data.fields['scan.interval']).toBe('Duración no válida');
		expect(result.data.values['scan.interval']).toBe('30s');
	});

	it('no llama a la API si el formulario no es válido', async () => {
		const response = settingsResponse();
		const fetchMock = sectionFetch(response);
		const result = (await save(
			actionEvent(fetchMock, 'save', {
				'scan.interval': 'mañana',
				'scan.on_startup': '1'
			}) as never
		)) as { status: number; data: { fields: Record<string, string> } };
		expect(result.status).toBe(400);
		expect(result.data.fields['scan.interval']).toBeTruthy();
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === 'PATCH')
		).toBe(false);
	});

	it('restablece un ajuste con DELETE', async () => {
		const response = settingsResponse();
		const fetchMock = sectionFetch(response, (url, init) => {
			if (url.pathname === '/api/v1/settings/scan.interval' && init?.method === 'DELETE') {
				return jsonResponse({ key: 'scan.interval', value: '6h' });
			}
			return undefined;
		});
		const result = (await reset(
			actionEvent(fetchMock, 'reset', { key: 'scan.interval' }) as never
		)) as { success?: boolean; message?: string };
		expect(result.success).toBe(true);
		expect(result.message).toBe('Ajuste restablecido.');
	});

	it('traduce los errores de las acciones', async () => {
		const response = settingsResponse();
		const broken = sectionFetch(response, (url, init) => {
			if (url.pathname === '/api/v1/settings' && init?.method === 'PATCH') {
				return jsonResponse({ error: 'fallo' }, 500);
			}
			return undefined;
		});
		const result = (await save(
			actionEvent(broken, 'save', {
				'scan.interval': '30m',
				'scan.on_startup': '1'
			}) as never
		)) as { status: number };
		expect(result.status).toBe(502);

		const forbidden = sectionFetch(response, (url, init) => {
			if (url.pathname === '/api/v1/settings/scan.interval' && init?.method === 'DELETE') {
				return jsonResponse({ error: 'no admin' }, 403);
			}
			return undefined;
		});
		const forbiddenResult = (await reset(
			actionEvent(forbidden, 'reset', { key: 'scan.interval' }) as never
		)) as { status: number };
		expect(forbiddenResult.status).toBe(403);

		const unauthorized = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));
		await expect(
			save(actionEvent(unauthorized, 'save', { 'scan.interval': '30m' }) as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('sistema y actualizaciones', () => {
	function systemFetch(
		checkEnabled: boolean,
		remote: () => Response | Promise<Response>
	) {
		return vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/settings') {
				return jsonResponse(settingsResponse(checkEnabled));
			}
			if (url.hostname === 'raw.githubusercontent.com') {
				return remote();
			}
			throw new Error(`URL inesperada: ${url}`);
		});
	}

	it('avisa de una versión nueva', async () => {
		const fetchMock = systemFetch(true, () => new Response('99.0.0\n', { status: 200 }));
		const result = (await checkUpdates(
			actionEvent(fetchMock, 'checkUpdates', {}, 'system') as never
		)) as { success?: boolean; update: { upToDate: boolean; message: string } };
		expect(result.success).toBe(true);
		expect(result.update.upToDate).toBe(false);
		expect(result.update.message).toContain('99.0.0');
	});

	it('indica que está al día', async () => {
		const fetchMock = systemFetch(true, () => new Response(`${webVersion}\n`, { status: 200 }));
		const result = (await checkUpdates(
			actionEvent(fetchMock, 'checkUpdates', {}, 'system') as never
		)) as { update: { upToDate: boolean } };
		expect(result.update.upToDate).toBe(true);
	});

	it('no consulta la red si el ajuste está desactivado', async () => {
		const fetchMock = systemFetch(false, () => new Response('99.0.0', { status: 200 }));
		const result = (await checkUpdates(
			actionEvent(fetchMock, 'checkUpdates', {}, 'system') as never
		)) as { status: number };
		expect(result.status).toBe(400);
		expect(
			fetchMock.mock.calls.some(([input]) => String(input).includes('raw.githubusercontent.com'))
		).toBe(false);
	});

	it('muestra un error claro si falla la consulta', async () => {
		const fetchMock = systemFetch(true, () => {
			throw new Error('sin conexión');
		});
		const result = (await checkUpdates(
			actionEvent(fetchMock, 'checkUpdates', {}, 'system') as never
		)) as { status: number; data: { message: string } };
		expect(result.status).toBe(502);
		expect(result.data.message).toContain('sin conexión');
	});

	it('renderiza las versiones y el botón solo con el ajuste activo', () => {
		const enabled = render(SettingsPage, {
			props: {
				data: {
					section: systemSection(true),
					serverInfo,
					webVersion
				} as never,
				form: {
					update: { latest: '99.0.0', current: webVersion, upToDate: false, message: 'Hay una versión nueva: 99.0.0' },
					command: './scripts/actualizar.sh'
				}
			}
		}).body;
		expect(enabled).toContain('Comprobar actualizaciones');
		expect(enabled).toContain('Hay una versión nueva: 99.0.0');
		expect(enabled).toContain('./scripts/actualizar.sh');
		expect(enabled).toContain('abc1234');

		const disabled = render(SettingsPage, {
			props: {
				data: { section: systemSection(false), serverInfo: null, webVersion } as never
			}
		}).body;
		expect(disabled).not.toContain('Comprobar actualizaciones');
		expect(disabled).toContain('conexión saliente opcional');
	});
});

describe('SettingsForm', () => {
	it('renderiza cada tipo de control, el defecto y los errores', () => {
		const settings = [
			...scanSection().settings,
			...metadataSection().settings,
			...uiSection().settings,
			...playbackSection().settings
		];
		const body = render(SettingsForm, {
			props: {
				settings,
				form: {
					fields: { 'playback.crf': 'El valor no es válido.' },
					values: { 'playback.crf': '99' }
				}
			}
		}).body;

		expect(body).toContain('type="checkbox"');
		expect(body).toContain('<select');
		expect(body).toContain('type="number"');
		expect(body).toContain('placeholder="6h, 30m"');
		expect(body).toContain('Por defecto: 6 h');
		expect(body).toContain('role="alert"');
		expect(body).toContain('El valor no es válido.');
		expect(body).toContain('Guardar cambios');
	});

	it('ofrece restablecer solo los ajustes modificados', () => {
		const modified = { ...scanSection().settings[0], modified: true };
		const body = render(SettingsForm, {
			props: { settings: [modified] }
		}).body;
		expect(body).toContain('Restablecer');
		expect(body).toContain('formaction="?/reset"');
		expect(body).toContain('name="key"');
	});
});
