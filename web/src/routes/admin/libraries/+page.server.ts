import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { ApiError, type LibraryKind } from '$lib/api';
import { createApiClient } from '$lib/api-factory';
import { requireSessionToken } from '$lib/session';

const ADMIN_MESSAGE = 'Esta sección es solo para administradores.';
const ALLOWED_KINDS = new Set<LibraryKind>(['video', 'clips', 'music']);

function actionFailure(
	reason: unknown,
	fallback: string,
	extra: Record<string, unknown> = {}
) {
	if (reason instanceof ApiError) {
		if (reason.status === 401) {
			redirect(303, '/login');
		}
		if (reason.status === 403) {
			return fail(403, { message: ADMIN_MESSAGE, ...extra });
		}
		if (reason.status === 400 || reason.status === 404 || reason.status === 409) {
			return fail(reason.status, { message: reason.message || fallback, ...extra });
		}
	}
	return fail(502, { message: fallback, ...extra });
}

function validName(name: string): boolean {
	const length = Array.from(name).length;
	return length >= 1 && length <= 80;
}

function validID(value: FormDataEntryValue | null): number | null {
	const id = Number(value);
	return Number.isInteger(id) && id > 0 ? id : null;
}

function formPaths(form: FormData): string[] {
	return form
		.getAll('paths')
		.map((value) => String(value).trim())
		.filter((value) => value !== '');
}

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const token = requireSessionToken(locals);
	const api = createApiClient(fetch);

	let user;
	try {
		user = await api.getCurrentUser(token);
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, 'No se pudo comprobar tu cuenta. Inténtalo de nuevo.');
	}

	if (user.role !== 'admin') {
		error(403, ADMIN_MESSAGE);
	}

	try {
		const { libraries } = await api.listLibraries(token);
		return { libraries };
	} catch (reason) {
		if (reason instanceof ApiError && reason.status === 401) {
			redirect(303, '/login');
		}
		error(502, 'No se pudieron cargar las bibliotecas. Inténtalo de nuevo.');
	}
};

export const actions: Actions = {
	create: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const name = String(form.get('name') ?? '').trim();
		const kindValue = String(form.get('kind') ?? 'video').trim();
		const kind = (kindValue || 'video') as LibraryKind;
		const paths = formPaths(form);
		const legacyPath = String(form.get('path') ?? '').trim();
		if (paths.length === 0 && legacyPath) {
			paths.push(legacyPath);
		}

		if (!validName(name)) {
			return fail(400, {
				message: 'El nombre debe tener entre 1 y 80 caracteres.',
				action: 'create',
				name,
				kind,
				paths,
				path: paths[0] ?? ''
			});
		}
		if (!ALLOWED_KINDS.has(kind)) {
			return fail(400, {
				message: 'Tipo de biblioteca no válido.',
				action: 'create',
				name,
				kind,
				paths,
				path: paths[0] ?? ''
			});
		}
		if (paths.length === 0) {
			return fail(400, {
				message: 'Añade al menos una carpeta.',
				action: 'create',
				name,
				kind,
				paths,
				path: ''
			});
		}

		try {
			await createApiClient(fetch).createLibrary({ name, kind, paths }, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo crear la biblioteca.', {
				action: 'create',
				name,
				kind,
				paths,
				path: paths[0] ?? ''
			});
		}

		return { success: true, message: 'Biblioteca creada.' };
	},

	rename: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		const name = String(form.get('name') ?? '').trim();

		if (id === null) {
			return fail(400, { message: 'Biblioteca no válida.', action: 'rename' });
		}
		if (!validName(name)) {
			return fail(400, {
				message: 'El nombre debe tener entre 1 y 80 caracteres.',
				action: 'rename',
				name,
				libraryId: id,
				renameName: name
			});
		}

		try {
			await createApiClient(fetch).renameLibrary(id, name, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo renombrar la biblioteca.', {
				action: 'rename',
				name,
				libraryId: id,
				renameName: name
			});
		}

		return { success: true, message: 'Biblioteca renombrada.' };
	},

	addPath: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		const path = String(form.get('path') ?? '').trim();

		if (id === null) {
			return fail(400, { message: 'Biblioteca no válida.', action: 'addPath' });
		}
		if (!path) {
			return fail(400, {
				message: 'Escribe una carpeta.',
				action: 'addPath',
				libraryId: id,
				path
			});
		}

		try {
			await createApiClient(fetch).addLibraryPath(id, path, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo añadir la carpeta.', {
				action: 'addPath',
				libraryId: id,
				path
			});
		}

		return { success: true, message: 'Carpeta añadida.' };
	},

	removePath: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		const pathId = validID(form.get('pathId'));

		if (id === null || pathId === null) {
			return fail(400, {
				message: 'Carpeta no válida.',
				action: 'removePath',
				libraryId: id ?? undefined,
				pathId: pathId ?? undefined
			});
		}

		try {
			await createApiClient(fetch).removeLibraryPath(id, pathId, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo quitar la carpeta.', {
				action: 'removePath',
				libraryId: id,
				pathId
			});
		}

		return { success: true, message: 'Carpeta quitada.' };
	},

	delete: async ({ request, locals, fetch }) => {
		const token = requireSessionToken(locals);
		const form = await request.formData();
		const id = validID(form.get('id'));
		const expectedName = String(form.get('name') ?? '').trim();
		const confirmation = String(form.get('confirm') ?? '').trim();

		if (id === null) {
			return fail(400, { message: 'Biblioteca no válida.', action: 'delete' });
		}
		if (expectedName && confirmation !== expectedName) {
			return fail(400, {
				message: `Escribe «${expectedName}» para confirmar.`,
				action: 'delete',
				libraryId: id
			});
		}

		try {
			await createApiClient(fetch).deleteLibrary(id, token);
		} catch (reason) {
			return actionFailure(reason, 'No se pudo borrar la biblioteca.', {
				action: 'delete',
				libraryId: id
			});
		}

		return { success: true, message: 'Biblioteca borrada.' };
	},

	scan: async ({ locals, fetch }) => {
		const token = requireSessionToken(locals);

		try {
			await createApiClient(fetch).startScan(token);
			return { success: true, started: true, message: 'Escaneo iniciado.' };
		} catch (reason) {
			if (reason instanceof ApiError && reason.status === 409) {
				return {
					success: true,
					started: true,
					message: 'Ya hay un escaneo en curso.'
				};
			}
			return actionFailure(reason, 'No se pudo iniciar el escaneo.');
		}
	}
};
