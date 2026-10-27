import type { FSBrowseResponse, LibraryKind } from './api';

export type LibraryAction =
	| 'create'
	| 'rename'
	| 'addPath'
	| 'removePath'
	| 'delete'
	| 'scan';

export interface LibraryFormState {
	message?: string;
	success?: boolean;
	action?: LibraryAction;
	libraryId?: number;
	name?: string;
	kind?: string;
	paths?: string[];
	path?: string;
	pathId?: number;
	renameName?: string;
	started?: boolean;
}

export interface Breadcrumb {
	name: string;
	path: string;
}

export interface LibraryKindOption {
	value: LibraryKind;
	label: string;
	disabled: boolean;
}

export const LIBRARY_KIND_LABELS: Record<LibraryKind, string> = {
	video: 'Películas y series',
	clips: 'Vídeos',
	music: 'Música'
};

export const LIBRARY_KIND_OPTIONS: LibraryKindOption[] = [
	{ value: 'video', label: 'Películas y series', disabled: false },
	{ value: 'clips', label: 'Vídeos (videoclips, conciertos…)', disabled: false },
	{ value: 'music', label: 'Música', disabled: true }
];

export function libraryKindLabel(kind: string | null | undefined): string {
	if (kind === 'video' || kind === 'clips' || kind === 'music') {
		return LIBRARY_KIND_LABELS[kind];
	}
	return 'Películas y series';
}

// Devuelve las migas desde la primera carpeta visible hasta la actual. En una
// ruta absoluta no se incluye la raíz del sistema porque el navegador solo
// puede moverse dentro de las raíces permitidas.
export function breadcrumbs(path: string): Breadcrumb[] {
	if (!path) {
		return [];
	}

	if (/^[A-Za-z]:[\\/]/.test(path) || path.startsWith('\\\\')) {
		return windowsBreadcrumbs(path);
	}

	const normalized = path.replace(/\\/g, '/').replace(/\/+$/, '');
	const parts = normalized.split('/').filter(Boolean);
	const crumbs: Breadcrumb[] = [];
	let current = '';

	for (const part of parts) {
		current += `/${part}`;
		crumbs.push({ name: part, path: current });
	}

	return crumbs;
}

export function parentPath(path: string): string | null {
	if (!path) {
		return null;
	}

	if (/^[A-Za-z]:[\\/]/.test(path) || path.startsWith('\\\\')) {
		return windowsParent(path);
	}

	let normalized = path.replace(/\\/g, '/').replace(/\/+$/, '');
	if (!normalized) {
		return null;
	}
	if (normalized === '/') {
		return null;
	}

	const index = normalized.lastIndexOf('/');
	if (index < 0) {
		return null;
	}
	normalized = normalized.slice(0, index);
	return normalized || '/';
}

export function browserParent(listing: FSBrowseResponse | null): string | null {
	return listing?.parent ?? null;
}

function windowsBreadcrumbs(path: string): Breadcrumb[] {
	const normalized = path.replace(/\//g, '\\').replace(/\\+$/, '');
	const drive = normalized.match(/^([A-Za-z]:)(\\|$)/);
	if (!drive) {
		const parts = normalized.split('\\').filter(Boolean);
		const crumbs: Breadcrumb[] = [];
		let current = '\\\\';
		for (const part of parts) {
			current = current === '\\\\' ? `${current}${part}` : `${current}\\${part}`;
			crumbs.push({ name: part, path: current });
		}
		return crumbs;
	}

	const root = `${drive[1]}\\`;
	const crumbs: Breadcrumb[] = [{ name: root, path: root }];
	const rest = normalized.slice(root.length).split('\\').filter(Boolean);
	let current = root;

	for (const part of rest) {
		current = current.endsWith('\\') ? `${current}${part}` : `${current}\\${part}`;
		crumbs.push({ name: part, path: current });
	}

	return crumbs;
}

function windowsParent(path: string): string | null {
	let normalized = path.replace(/\//g, '\\').replace(/\\+$/, '');
	if (/^[A-Za-z]:$/.test(normalized)) {
		normalized += '\\';
	}
	if (/^[A-Za-z]:\\$/.test(normalized)) {
		return null;
	}

	const index = normalized.lastIndexOf('\\');
	if (index < 0) {
		return null;
	}
	const parent = normalized.slice(0, index);
	if (/^[A-Za-z]:$/.test(parent)) {
		return `${parent}\\`;
	}
	return parent || null;
}
