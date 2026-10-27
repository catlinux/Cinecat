import type { VideoBreadcrumb } from './api';

export type ParsedVideoParam<T> = { ok: true; value: T } | { ok: false };

export interface VideoBreadcrumbLink {
	name: string;
	href: string;
}

export function parseVideoLibraryID(raw: string | null | undefined): number | null {
	const value = Number(raw);
	return Number.isInteger(value) && value > 0 ? value : null;
}

export function parseVideoRoot(raw: string | null | undefined): ParsedVideoParam<number | null> {
	const text = String(raw ?? '').trim();
	if (text === '') {
		return { ok: true, value: null };
	}
	const value = Number(text);
	if (!Number.isInteger(value) || value <= 0) {
		return { ok: false };
	}
	return { ok: true, value };
}

export function parseVideoPath(
	raw: string | null | undefined,
	hasRoot: boolean
): ParsedVideoParam<string> {
	const path = String(raw ?? '');
	if (path === '') {
		return { ok: true, value: '' };
	}
	if (!hasRoot) {
		return { ok: false };
	}
	if (
		path.startsWith('/') ||
		path.startsWith('\\') ||
		/^[A-Za-z]:/.test(path) ||
		path.includes('\\')
	) {
		return { ok: false };
	}
	const parts = path.split('/');
	for (const part of parts) {
		if (part === '' || part === '.' || part === '..') {
			return { ok: false };
		}
	}
	return { ok: true, value: path };
}

export function videoRootHref(libraryID: number, rootID: number): string {
	return `/videos/${libraryID}?root=${rootID}`;
}

export function videoFolderHref(
	libraryID: number,
	rootID: number,
	path: string
): string {
	const params = new URLSearchParams({ root: String(rootID) });
	if (path) {
		params.set('path', path);
	}
	return `/videos/${libraryID}?${params}`;
}

export function videoBreadcrumbs(
	libraryID: number,
	libraryName: string,
	root: { id: number; name: string } | null,
	crumbs: VideoBreadcrumb[]
): VideoBreadcrumbLink[] {
	const result: VideoBreadcrumbLink[] = [
		{ name: 'Vídeos', href: '/videos' },
		{ name: libraryName || 'Biblioteca', href: `/videos/${libraryID}` }
	];
	if (root) {
		for (const crumb of crumbs) {
			const params = new URLSearchParams({ root: String(root.id) });
			if (crumb.path) {
				params.set('path', crumb.path);
			}
			result.push({ name: crumb.name || root.name, href: `/videos/${libraryID}?${params}` });
		}
	}
	return result;
}

export function videoCountLabel(count: number): string {
	const value = Number.isFinite(count) && count > 0 ? Math.floor(count) : 0;
	return `${value} ${value === 1 ? 'vídeo' : 'vídeos'}`;
}

export function videoHeading(
	path: string,
	crumbs: VideoBreadcrumb[],
	fallback: string
): string {
	if (path && crumbs.length > 0) {
		return crumbs[crumbs.length - 1].name || fallback;
	}
	return fallback;
}
