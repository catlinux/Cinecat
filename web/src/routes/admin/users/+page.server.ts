import { error, fail, redirect } from '@sveltejs/kit';
import {
	ApiError,
	type ManagedUser,
	type Session,
	type UpdateUserInput,
	type UserRole
} from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { Actions, PageServerLoad } from './$types';

const ADMIN_MESSAGE = 'Esta sección es solo para administradores.';
const LOAD_ERROR = 'No se pudieron cargar los usuarios. Inténtalo de nuevo.';
const ROLES = new Set<UserRole>(['admin', 'user']);

type ManagedUserWithSessions = ManagedUser & { sessions: Session[] | null };

function actionFailure(reason: unknown, fallback: string, extra: Record<string, unknown> = {}) {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason.status === 400 || reason.status === 403 || reason.status === 404 || reason.status === 409) {
			return fail(reason.status, { message: reason.message || fallback, ...extra });
		}
	}
	return fail(502, { message: fallback, ...extra });
}

function validID(value: FormDataEntryValue | null): number | null {
	const id = Number(value);
	return Number.isInteger(id) && id > 0 ? id : null;
}

function readLibraryIDs(form: FormData): number[] {
	const seen = new Set<number>();
	const ids: number[] = [];
	for (const value of form.getAll('library_ids')) {
		const id = Number(value);
		if (!Number.isInteger(id) || id <= 0 || seen.has(id)) {
			continue;
		}
		seen.add(id);
		ids.push(id);
	}
	return ids;
}

function readBool(value: FormDataEntryValue | null, fallback = false): boolean {
	if (value === null) {
		return fallback;
	}
	const normalized = String(value).trim().toLowerCase();
	if (['true', '1', 'on', 'yes'].includes(normalized)) {
		return true;
	}
	if (['false', '0', 'off', 'no'].includes(normalized)) {
		return false;
	}
	return fallback;
}

function createFormState(form: FormData) {
	return {
		username: String(form.get('username') ?? '').trim(),
		role: String(form.get('role') ?? 'user'),
		all_libraries: readBool(form.get('all_libraries')),
		library_ids: readLibraryIDs(form)
	};
}

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const token = requireSessionToken(locals);
	const api = createApiClient(fetch);

	let currentUser;
	try {
		currentUser = await api.getCurrentUser(token);
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, 'No se pudo comprobar tu cuenta. Inténtalo de nuevo.');
	}
	if (currentUser.role !== 'admin') {
		error(403, ADMIN_MESSAGE);
	}

	try {
		const [usersResponse, librariesResponse] = await Promise.all([
			api.listUsers(token),
			api.listLibraries(token)
		]);
		const users: ManagedUserWithSessions[] = await Promise.all(
			(usersResponse.users ?? []).map(async (user) => {
				try {
					const response = await api.getUserSessions(user.id, token);
					return { ...user, sessions: response.sessions ?? [] };
				} catch {
					// La tarjeta del usuario sigue funcionando sin sus sesiones.
					return { ...user, sessions: null };
				}
			})
		);
		return {
			users,
			libraries: librariesResponse.libraries ?? [],
			currentUser
		};
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, LOAD_ERROR);
	}
};

export const actions: Actions = {
	create: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const state = createFormState(form);
		const password = String(form.get('password') ?? '');
		const repeatPassword = String(form.get('repeat_password') ?? '');
		const role = state.role as UserRole;
		const libraryIDs = state.library_ids;

		if (Array.from(state.username).length < 1 || Array.from(state.username).length > 40) {
			return fail(400, {
				message: 'El nombre debe tener entre 1 y 40 caracteres.',
				action: 'create',
				create: state
			});
		}
		if (Array.from(password).length < 8) {
			return fail(400, {
				message: 'La contraseña debe tener al menos 8 caracteres.',
				action: 'create',
				create: state
			});
		}
		if (password !== repeatPassword) {
			return fail(400, {
				message: 'Las contraseñas no coinciden.',
				action: 'create',
				create: state
			});
		}
		if (!ROLES.has(role)) {
			return fail(400, { message: 'Rol no válido.', action: 'create', create: state });
		}
		if (role === 'user' && !state.all_libraries && libraryIDs.length === 0) {
			return fail(400, {
				message: 'Elige al menos una biblioteca o marca «Todas».',
				action: 'create',
				create: state
			});
		}

		try {
			await createApiClient(fetch).createUser(
				{
					username: state.username,
					password,
					role,
					all_libraries: state.all_libraries,
					library_ids: libraryIDs
				},
				token
			);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo crear el usuario.', {
				action: 'create',
				create: state
			});
		}

		return { success: true, message: 'Usuario creado.', action: 'create' };
	},

	update: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		if (id === null) {
			return fail(400, { message: 'Usuario no válido.', action: 'update' });
		}

		const changes: UpdateUserInput = {};
		if (form.has('role')) {
			const role = String(form.get('role') ?? '').trim() as UserRole;
			if (!ROLES.has(role)) {
				return fail(400, { message: 'Rol no válido.', action: 'update', userId: id });
			}
			changes.role = role;
		}
		if (form.has('disabled')) {
			changes.disabled = readBool(form.get('disabled'));
		}
		if (form.has('all_libraries') || form.has('all_libraries_present')) {
			changes.all_libraries = readBool(form.get('all_libraries'));
		}
		if (Object.keys(changes).length === 0) {
			return fail(400, {
				message: 'No hay ningún cambio que guardar.',
				action: 'update',
				userId: id
			});
		}

		try {
			await createApiClient(fetch).updateUser(id, changes, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo actualizar el usuario.', {
				action: 'update',
				userId: id
			});
		}

		return { success: true, message: 'Usuario actualizado.', action: 'update', userId: id };
	},

	libraries: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		if (id === null) {
			return fail(400, { message: 'Usuario no válido.', action: 'libraries' });
		}
		const libraryIDs = readLibraryIDs(form);
		const allLibrariesPresent = form.has('all_libraries_present') || form.has('all_libraries');
		const allLibraries = readBool(form.get('all_libraries'));

		try {
			const api = createApiClient(fetch);
			await api.setUserLibraries(id, libraryIDs, token);
			if (allLibrariesPresent) {
				await api.updateUser(id, { all_libraries: allLibraries }, token);
			}
		} catch (reason) {
			return actionFailure(reason, 'No se pudieron guardar las bibliotecas.', {
				action: 'libraries',
				userId: id
			});
		}

		return {
			success: true,
			message: 'Bibliotecas actualizadas.',
			action: 'libraries',
			userId: id
		};
	},

	password: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		const password = String(form.get('password') ?? '');
		const repeatPassword = String(form.get('repeat_password') ?? '');
		if (id === null) {
			return fail(400, { message: 'Usuario no válido.', action: 'password' });
		}
		if (Array.from(password).length < 8) {
			return fail(400, {
				message: 'La contraseña debe tener al menos 8 caracteres.',
				action: 'password',
				userId: id
			});
		}
		if (password !== repeatPassword) {
			return fail(400, {
				message: 'Las contraseñas no coinciden.',
				action: 'password',
				userId: id
			});
		}

		try {
			await createApiClient(fetch).resetUserPassword(id, password, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo restablecer la contraseña.', {
				action: 'password',
				userId: id
			});
		}

		return {
			success: true,
			message: 'Contraseña restablecida. Se han cerrado las sesiones de la cuenta.',
			action: 'password',
			userId: id
		};
	},

	delete: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		const confirmation = String(form.get('confirmation') ?? '').trim();
		if (id === null) {
			return fail(400, { message: 'Usuario no válido.', action: 'delete' });
		}

		let target: ManagedUser | undefined;
		try {
			const response = await createApiClient(fetch).listUsers(token);
			target = (response.users ?? []).find((user) => user.id === id);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo comprobar el usuario.', {
				action: 'delete',
				userId: id
			});
		}
		if (!target) {
			return fail(404, { message: 'Usuario no encontrado.', action: 'delete', userId: id });
		}
		if (confirmation !== target.username) {
			return fail(400, {
				message: 'Escribe exactamente el nombre del usuario para confirmar.',
				action: 'delete',
				userId: id
			});
		}

		try {
			await createApiClient(fetch).deleteUser(id, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo eliminar el usuario.', {
				action: 'delete',
				userId: id
			});
		}

		return { success: true, message: 'Usuario eliminado.', action: 'delete', userId: id };
	},

	closeSession: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const userID = validID(form.get('user_id'));
		const sessionID = validID(form.get('session_id'));
		if (userID === null || sessionID === null) {
			return fail(400, {
				message: 'Sesión no válida.',
				action: 'closeSession',
				userId: userID ?? undefined
			});
		}

		try {
			await createApiClient(fetch).closeUserSession(userID, sessionID, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo cerrar la sesión.', {
				action: 'closeSession',
				userId: userID
			});
		}

		return {
			success: true,
			message: 'Sesión cerrada.',
			action: 'closeSession',
			userId: userID
		};
	}
};
