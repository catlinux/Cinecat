import { error, fail, redirect } from '@sveltejs/kit';
import type { ServerVersion, SettingsSection, UpdateInfo } from '$lib/api';
import { ApiError } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import {
	collectSettingsForm,
	findSettingsSection,
	isSettingsSection
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
	}

	return { section, serverInfo, updateInfo, webVersion };
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
	}
};
