import type { TrashItem } from './api';

export function trashDaysRemaining(expiresAt: string, now = Date.now()): number {
	const deadline = Date.parse(expiresAt);
	if (Number.isNaN(deadline)) {
		return 0;
	}
	return Math.max(0, Math.ceil((deadline - now) / 86_400_000));
}

export function trashKindLabel(item: Pick<TrashItem, 'kind' | 'moved_kind'>): string {
	const kind =
		item.kind === 'movie'
			? 'Película'
			: item.kind === 'episode'
				? 'Episodio'
				: item.kind === 'series'
					? 'Serie'
					: item.kind;
	const moved = item.moved_kind === 'folder' ? 'carpeta' : 'archivo';
	return `${kind} · ${moved}`;
}

export function trashExpiryLabel(expiresAt: string, now = Date.now()): string {
	const days = trashDaysRemaining(expiresAt, now);
	if (days === 0) {
		return 'Caducado';
	}
	return days === 1 ? 'Queda 1 día' : `Quedan ${days} días`;
}
