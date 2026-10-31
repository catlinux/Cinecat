import { error, redirect } from '@sveltejs/kit';
import {
	ApiError,
	type ActivityEntry,
	type ManagedUser,
	type UserRole
} from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';
import type { PageServerLoad } from './$types';

const ADMIN_MESSAGE = 'Esta sección es solo para administradores.';
const LOAD_ERROR = 'No se pudo cargar la actividad. Inténtalo de nuevo.';
const PAGE_LIMIT = 100;

function readUserFilter(url: URL): number | null {
	const raw = (url.searchParams.get('user_id') ?? '').trim();
	if (!raw) {
		return null;
	}
	const value = Number(raw);
	return Number.isInteger(value) && value > 0 ? value : null;
}

export const load: PageServerLoad = async ({ locals, fetch, url }) => {
	const token = requireSessionToken(locals);
	const api = createApiClient(fetch);

	let currentUser: { username: string; role: UserRole };
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

	const userFilter = readUserFilter(url);
	if (userFilter === null && (url.searchParams.get('user_id') ?? '').trim() !== '') {
		error(400, 'El filtro de usuario no es válido.');
	}

	let users: ManagedUser[];
	let entries: ActivityEntry[];
	try {
		const [usersResponse, activity] = await Promise.all([
			api.listUsers(token),
			api.getActivity(
				userFilter === null
					? { limit: PAGE_LIMIT }
					: { limit: PAGE_LIMIT, user_id: userFilter },
				token
			)
		]);
		users = usersResponse.users ?? [];
		entries = (activity.entries ?? []) as ActivityEntry[];
	} catch (reason) {
		if (reason instanceof ApiError) {
			if (reason.status === 401) {
				redirect(303, '/login');
			}
			if (reason.status === 403) {
				error(403, ADMIN_MESSAGE);
			}
		}
		error(502, LOAD_ERROR);
	}
	if (userFilter !== null && !users.some((user) => user.id === userFilter)) {
		error(404, 'Usuario no encontrado.');
	}
	return {
		entries,
		users,
		selectedUserId: userFilter,
		limit: PAGE_LIMIT,
		currentUser
	};
};
