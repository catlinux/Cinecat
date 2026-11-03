import type { PhotoSort } from './api';

export const PHOTO_SORTS: PhotoSort[] = ['date', 'name'];

export const PHOTO_SORT_LABELS: Record<PhotoSort, string> = {
	date: 'Fecha',
	name: 'Nombre'
};

export interface PhotoBreadcrumb {
	name: string;
	path: string;
}

export function parsePhotoSort(raw: unknown): PhotoSort {
	const value = String(raw ?? '').trim().toLowerCase();
	return value === 'name' ? 'name' : 'date';
}

// La carpeta llega como ruta relativa a la raíz de la biblioteca. Se rechazan
// rutas absolutas y componentes que pudieran salir de ella.
export function parsePhotoFolder(raw: unknown): string | null {
	let value = String(raw ?? '').trim();
	if (value === '') {
		return '';
	}
	value = value.replace(/\\/g, '/');
	if (value.startsWith('/') || /^[A-Za-z]:/.test(value) || value.startsWith('//')) {
		return null;
	}
	const parts = value.split('/');
	if (parts.some((part) => part === '' || part === '.' || part === '..')) {
		return null;
	}
	return parts.join('/');
}

export function photoBreadcrumbs(folder: string): PhotoBreadcrumb[] {
	if (!folder) {
		return [];
	}
	const crumbs: PhotoBreadcrumb[] = [];
	let current = '';
	for (const part of folder.split('/').filter(Boolean)) {
		current = current ? `${current}/${part}` : part;
		crumbs.push({ name: part, path: current });
	}
	return crumbs;
}

export function photoLibraryHref(libraryID: number): string {
	return `/photos/${libraryID}`;
}

export function photoFolderHref(
	libraryID: number,
	folder: string,
	sort: PhotoSort = 'date'
): string {
	const params = new URLSearchParams();
	if (folder) {
		params.set('folder', folder);
	}
	if (sort !== 'date') {
		params.set('sort', sort);
	}
	const suffix = params.toString();
	return `${photoLibraryHref(libraryID)}${suffix ? `?${suffix}` : ''}`;
}

export function photoThumbURL(id: number): string {
	return `/photos/thumb/${id}`;
}

export function photoOriginalURL(id: number): string {
	return `/photos/original/${id}`;
}

export function nextPhotoIndex(index: number, total: number): number {
	if (!Number.isInteger(index) || !Number.isFinite(total) || total <= 0 || index < 0) {
		return 0;
	}
	if (index >= total) {
		return 0;
	}
	return (index + 1) % total;
}

export function previousPhotoIndex(index: number, total: number): number {
	if (!Number.isInteger(index) || !Number.isFinite(total) || total <= 0 || index < 0) {
		return 0;
	}
	if (index >= total) {
		return total - 1;
	}
	return (index - 1 + total) % total;
}

export function photoCountLabel(total: number): string {
	const count = Number.isFinite(total) && total > 0 ? Math.floor(total) : 0;
	return `${count} ${count === 1 ? 'foto' : 'fotos'}`;
}

export function photoDateLabel(takenAt: number | null | undefined): string {
	if (takenAt === null || takenAt === undefined || !Number.isFinite(takenAt)) {
		return '';
	}
	const date = new Date(takenAt * 1000);
	if (Number.isNaN(date.getTime())) {
		return '';
	}
	return date.toLocaleDateString('es-ES', {
		day: 'numeric',
		month: 'short',
		year: 'numeric'
	});
}

export function photoAspectRatio(width: number, height: number): string {
	if (Number.isFinite(width) && Number.isFinite(height) && width > 0 && height > 0) {
		return `${width} / ${height}`;
	}
	return '4 / 3';
}
