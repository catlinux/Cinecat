import { error, fail, redirect } from '@sveltejs/kit';
import type { Backup, BackupsResponse, ServerVersion, SettingsSection, UpdateInfo } from '$lib/api';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import {
	collectSettingsForm,
	findSettingsSection,
	isSettingsSection,
	parsePortalRows,
	type PortalRow
} from '$lib/settings';
import { webVersion } from '$lib/version';
import type { Actions, PageServerLoad } from './$types';

const ADMIN_MESSAGE = 'Esta sección es solo para administradores.';
const LOAD_ERROR = 'No se pudieron cargar los ajustes. Inténtalo de nuevo.';
const UPDATE_CHECK_ERROR = 'No se pudo comprobar si hay una versión nueva.';
const UPDATE_APPLY_ERROR = 'No se pudo solicitar la actualización.';
const UPDATE_OBSOLETE = 'Esa versión ya no es la última: vuelve a comprobar';
const UPDATE_RUNNING = 'Ya hay una actualización en curso';
const UPDATE_UNSUPPORTED =
	'La actualización automática no está instalada en este servidor';
const BACKUP_CREATE_ERROR = 'No se pudo crear la copia de seguridad.';
const BACKUP_RUNNING = 'Ya hay una copia en curso';

function sectionError(reason: unknown, fallback = LOAD_ERROR): never {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason.status === 403) {
			error(403, ADMIN_MESSAGE);
		}
		if (reason.status === 404) {
			error(404, reason.message || 'Sección de ajustes no encontrada.');
		}
	}
	error(502, fallback);
}

function actionFailure(reason: unknown, fallback: string) {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason.status === 403) {
			return fail(403, { message: ADMIN_MESSAGE });
		}
		if (reason.status === 400) {
			return fail(400, {
				message: reason.message || fallback,
				fields: reason.fields ?? {}
			});
		}
		if (reason.status === 404) {
			return fail(404, { message: reason.message || fallback });
		}
	}
	return fail(502, { message: fallback });
}

async function loadSettingsSection(
	fetchImpl: typeof fetch,
	token: string,
	sectionKey: string
): Promise<SettingsSection> {
	const api = createApiClient(fetchImpl);
	const response = await api.getSettings(token);
	const section = findSettingsSection(response, sectionKey);
	if (!section) {
		error(404, 'Sección de ajustes no encontrada.');
	}
	return section;
}

function readPortalRows(form: FormData): PortalRow[] | null {
	const countRaw = form.get('row_count');
	if (countRaw !== null && Number.isInteger(Number(countRaw)) && Number(countRaw) >= 0) {
		const count = Number(countRaw);
		const rows: PortalRow[] = [];
		for (let index = 0; index < count; index += 1) {
			const id = String(form.get(`row_id_${index}`) ?? '').trim();
			if (!id) {
				return null;
			}
			rows.push({
				id,
				visible: ['1', 'true', 'on'].includes(
					String(form.get(`row_visible_${index}`) ?? '').toLowerCase()
				),
				title: String(form.get(`row_title_${index}`) ?? ''),
				kicker: String(form.get(`row_kicker_${index}`) ?? '')
			});
		}
		return parsePortalRows(rows);
	}

	const raw = String(
		form.get('portal_rows') ?? form.get('rows') ?? form.get('ui.portal_rows') ?? ''
	).trim();
	if (!raw) {
		return null;
	}
	try {
		const parsed: unknown = JSON.parse(raw);
		if (!Array.isArray(parsed)) {
			return null;
		}
		return parsePortalRows(parsed);
	} catch {
		return null;
	}
}

function portalRowsFailure(reason: unknown, rows: PortalRow[]) {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason.status === 403) {
			return fail(403, {
				portalRowsAction: true,
				message: ADMIN_MESSAGE,
				portalRows: rows
			});
		}
		if (reason.status === 400) {
			return fail(400, {
				portalRowsAction: true,
				message: reason.message || 'Las filas del portal no son válidas.',
				portalRows: rows
			});
		}
	}
	return fail(502, {
		portalRowsAction: true,
		message: 'No se pudieron guardar las filas del portal.',
		portalRows: rows
	});
}

export const load: PageServerLoad = async ({ params, locals, fetch }) => {
	const sectionKey = params.section;
	if (!isSettingsSection(sectionKey)) {
		error(404, 'Sección de ajustes no encontrada.');
	}

	const token = requireSessionToken(locals);
	let section: SettingsSection;
	try {
		section = await loadSettingsSection(fetch, token, sectionKey);
	} catch (reason) {
		sectionError(reason);
	}

	let serverInfo: ServerVersion | null = null;
	let updateInfo: UpdateInfo | null = null;
	let backupInfo: BackupsResponse | null = null;
	let backupLoadError = false;
	if (sectionKey === 'system') {
		const api = createApiClient(fetch);
		try {
			serverInfo = await api.getServerVersion();
		} catch {
			serverInfo = null;
		}
		try {
			updateInfo = await api.getUpdate(token);
		} catch {
			// La tarjeta de versiones sigue siendo útil sin el estado de actualización.
			updateInfo = null;
		}
		try {
			backupInfo = await api.getBackups(token);
		} catch {
			// Las copias no deben impedir que se vean los ajustes ni las versiones.
			backupInfo = null;
			backupLoadError = true;
		}
	}

	return { section, serverInfo, updateInfo, backupInfo, backupLoadError, webVersion };
};

export const actions: Actions = {
	save: async ({ request, locals, fetch, params }) => {
		const sectionKey = params.section;
		if (!isSettingsSection(sectionKey)) {
			return fail(404, { message: 'Sección de ajustes no encontrada.' });
		}
		const token = requireSessionToken(locals);

		let section: SettingsSection;
		try {
			section = await loadSettingsSection(fetch, token, sectionKey);
		} catch (reason) {
			return actionFailure(reason, LOAD_ERROR);
		}

		const formData = await request.formData();
		if (formData.get('__settings_form') !== '1') {
			for (const setting of section.settings) {
				if (setting.type === 'bool' && !formData.has(setting.key)) {
					formData.set(setting.key, setting.value ? '1' : '0');
				}
			}
		}
		const collected = collectSettingsForm(section.settings, formData);
		if (Object.keys(collected.fields).length > 0) {
			return fail(400, {
				message: 'Revisa los campos marcados.',
				fields: collected.fields,
				values: collected.values
			});
		}

		if (Object.keys(collected.changes).length > 0) {
			try {
				await createApiClient(fetch).patchSettings(collected.changes, token);
			} catch (reason) {
				if (reason instanceof ApiError && reason.status === 400) {
					return fail(400, {
						message: reason.message || 'Revisa los campos marcados.',
						fields: reason.fields ?? {},
						values: collected.values
					});
				}
				return actionFailure(reason, 'No se pudieron guardar los ajustes.');
			}
		}

		return {
			success: true,
			message: 'Ajustes guardados.'
		};
	},

	portalRows: async ({ request, locals, fetch, params }) => {
		if (params.section !== 'ui') {
			return fail(404, { message: 'Sección de ajustes no encontrada.' });
		}
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const rows = readPortalRows(form);
		if (!rows) {
			return fail(400, {
				portalRowsAction: true,
				message: 'Las filas del portal no son válidas.'
			});
		}

		const move = String(form.get('move') ?? '').trim();
		const [direction, moveId] = move.includes(':')
			? [move.slice(0, move.indexOf(':')), move.slice(move.indexOf(':') + 1)]
			: [move, String(form.get('move_id') ?? form.get('id') ?? '')];
		if ((direction === 'up' || direction === 'down') && moveId) {
			const index = rows.findIndex((row) => row.id === moveId);
			const target = direction === 'up' ? index - 1 : index + 1;
			if (index >= 0 && target >= 0 && target < rows.length) {
				[rows[index], rows[target]] = [rows[target], rows[index]];
			}
			return {
				portalRowsAction: true,
				success: true,
				message: 'Orden actualizado. Guarda los cambios.',
				portalRows: rows
			};
		}

		try {
			await createApiClient(fetch).patchSettings(
				{ 'ui.portal_rows': JSON.stringify(rows) },
				token
			);
			return {
				portalRowsAction: true,
				success: true,
				message: 'Filas del portal guardadas.',
				portalRows: rows
			};
		} catch (reason) {
			return portalRowsFailure(reason, rows);
		}
	},

	reset: async ({ request, locals, fetch, params }) => {
		const sectionKey = params.section;
		if (!isSettingsSection(sectionKey)) {
			return fail(404, { message: 'Sección de ajustes no encontrada.' });
		}
		const token = requireSessionToken(locals);
		const key = String((await request.formData()).get('key') ?? '').trim();
		if (!key) {
			return fail(400, { message: 'Ajuste no válido.' });
		}

		try {
			await createApiClient(fetch).resetSetting(key, token);
		} catch (reason) {
			if (reason instanceof ApiError && reason.status === 404) {
				return fail(404, { message: reason.message || 'Ajuste no encontrado.' });
			}
			return actionFailure(reason, 'No se pudo restablecer el ajuste.');
		}

		return { success: true, message: 'Ajuste restablecido.' };
	},

	checkUpdates: async ({ locals, fetch, params }) => {
		if (params.section !== 'system') {
			return fail(404, { message: 'Sección de ajustes no encontrada.' });
		}
		const token = requireSessionToken(locals);

		try {
			const updateInfo = await createApiClient(fetch).checkUpdate(token);
			return { updateAction: true, success: true, updateInfo };
		} catch (reason) {
			if (reason instanceof ApiError) {
				if (reason.status === 401) {
					redirect(303, '/login');
				}
				if (reason.status === 403) {
					return fail(403, { updateAction: true, message: ADMIN_MESSAGE });
				}
			}
			return fail(502, { updateAction: true, message: UPDATE_CHECK_ERROR });
		}
	},

	applyUpdate: async ({ request, locals, fetch, params }) => {
		if (params.section !== 'system') {
			return fail(404, { message: 'Sección de ajustes no encontrada.' });
		}
		const token = requireSessionToken(locals);
		const tag = String((await request.formData()).get('tag') ?? '').trim();
		if (!tag) {
			return fail(400, { updateAction: true, message: UPDATE_OBSOLETE });
		}

		try {
			const updateInfo = await createApiClient(fetch).applyUpdate(tag, token);
			return { updateAction: true, success: true, updateInfo, appliedTag: tag };
		} catch (reason) {
			if (reason instanceof ApiError) {
				if (reason.status === 401) {
					redirect(303, '/login');
				}
				if (reason.status === 403) {
					return fail(403, { updateAction: true, message: ADMIN_MESSAGE });
				}
				if (reason.status === 400) {
					return fail(400, { updateAction: true, message: UPDATE_OBSOLETE });
				}
				if (reason.status === 409) {
					return fail(409, { updateAction: true, message: UPDATE_RUNNING });
				}
				if (reason.status === 503) {
					return fail(503, {
						updateAction: true,
						message: UPDATE_UNSUPPORTED,
						installHelp: true
					});
				}
			}
			return fail(502, { updateAction: true, message: UPDATE_APPLY_ERROR });
		}
	},

	backup: async ({ locals, fetch, params }) => {
		if (params.section !== 'system') {
			return fail(404, { message: 'Sección de ajustes no encontrada.' });
		}
		const token = requireSessionToken(locals);

		let backup: Backup;
		try {
			backup = await createApiClient(fetch).createBackup(token);
		} catch (reason) {
			if (reason instanceof ApiError) {
				if (reason.status === 401) {
					redirect(303, '/login');
				}
				if (reason.status === 403) {
					return fail(403, { backupAction: true, message: ADMIN_MESSAGE });
				}
				if (reason.status === 409) {
					return fail(409, { backupAction: true, message: BACKUP_RUNNING });
				}
			}
			return fail(502, { backupAction: true, message: BACKUP_CREATE_ERROR });
		}

		return {
			backupAction: true,
			success: true,
			message: 'Copia de seguridad creada.',
			backup
		};
	}
};
