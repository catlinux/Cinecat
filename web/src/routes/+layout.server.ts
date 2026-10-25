import type { LayoutServerLoad } from './$types';

// Expone en el layout si hay sesión para mostrar el acceso a la cuenta.
export const load: LayoutServerLoad = ({ locals }) => {
	return { hasSession: locals.sessionToken !== null };
};
