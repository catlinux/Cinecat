export function pageWindow(current: number, last: number): Array<number | null> {
	const lastPage = Math.max(1, Math.floor(Number(last) || 1));
	const currentPage = Math.min(lastPage, Math.max(1, Math.floor(Number(current) || 1)));

	if (lastPage <= 7) {
		return Array.from({ length: lastPage }, (_, index) => index + 1);
	}

	const pages = new Set<number>([1, lastPage]);
	for (let page = currentPage - 2; page <= currentPage + 2; page++) {
		if (page >= 1 && page <= lastPage) {
			pages.add(page);
		}
	}

	const sorted = [...pages].sort((a, b) => a - b);
	const result: Array<number | null> = [];
	for (let index = 0; index < sorted.length; index++) {
		const page = sorted[index];
		if (index > 0 && page - sorted[index - 1] > 1) {
			result.push(null);
		}
		result.push(page);
	}
	return result;
}

export function clampPage(raw: unknown, last: number): number {
	const lastPage = Number.isFinite(last) && last >= 1 ? Math.floor(last) : 1;
	let value: number;
	if (typeof raw === 'number') {
		value = Math.trunc(raw);
	} else {
		value = Number.parseInt(String(raw ?? '').trim(), 10);
	}
	if (!Number.isFinite(value) || value < 1) {
		return 1;
	}
	return Math.min(value, lastPage);
}

export function pageCount(total: number, size: number): number {
	if (!Number.isFinite(total) || total <= 0 || !Number.isFinite(size) || size <= 0) {
		return 1;
	}
	return Math.max(1, Math.ceil(total / size));
}

export function pageHref(url: URL | string, page: number): string {
	const target =
		typeof url === 'string' ? new URL(url, 'http://cinecat.local') : new URL(url.toString());
	const value = Number.isFinite(page) && page >= 1 ? Math.floor(page) : 1;
	target.searchParams.set('page', String(value));
	return `${target.pathname}${target.search}`;
}
