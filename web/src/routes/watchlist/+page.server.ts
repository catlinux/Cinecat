import { loadUserListPage } from '$lib/lists-load';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => loadUserListPage('watchlist', event);
