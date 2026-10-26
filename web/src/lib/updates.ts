export const UPDATE_VERSION_URL =
	'https://raw.githubusercontent.com/catlinux/Cinecat/main/VERSION';

export const UPDATE_COMMAND = './scripts/actualizar.sh';

export interface ParsedVersion {
	major: number;
	minor: number;
	patch: number;
	prerelease: Array<string | number>;
}

export interface UpdateCheck {
	latest: string;
	current: string;
	upToDate: boolean;
	message: string;
}

export function parseVersion(raw: unknown): ParsedVersion | null {
	const text = String(raw ?? '').trim();
	const match =
		/^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/.exec(
			text
		);
	if (!match) {
		return null;
	}

	const numeric = [match[1], match[2], match[3]];
	if (numeric.some((value) => value.length > 1 && value.startsWith('0'))) {
		return null;
	}

	const rawPrerelease = match[4] ? match[4].split('.') : [];
	if (rawPrerelease.some((identifier) => /^0\d+$/.test(identifier))) {
		return null;
	}
	const prerelease = rawPrerelease.map((identifier) =>
		/^\d+$/.test(identifier) ? Number(identifier) : identifier
	);

	return {
		major: Number(match[1]),
		minor: Number(match[2]),
		patch: Number(match[3]),
		prerelease
	};
}

export function formatVersion(version: ParsedVersion): string {
	const base = `${version.major}.${version.minor}.${version.patch}`;
	return version.prerelease.length > 0 ? `${base}-${version.prerelease.join('.')}` : base;
}

export function compareVersions(left: string, right: string): number | null {
	const first = parseVersion(left);
	const second = parseVersion(right);
	if (!first || !second) {
		return null;
	}

	for (const key of ['major', 'minor', 'patch'] as const) {
		if (first[key] !== second[key]) {
			return first[key] > second[key] ? 1 : -1;
		}
	}

	if (first.prerelease.length === 0 && second.prerelease.length === 0) {
		return 0;
	}
	if (first.prerelease.length === 0) {
		return 1;
	}
	if (second.prerelease.length === 0) {
		return -1;
	}

	const length = Math.max(first.prerelease.length, second.prerelease.length);
	for (let index = 0; index < length; index += 1) {
		const leftPart = first.prerelease[index];
		const rightPart = second.prerelease[index];
		if (leftPart === undefined) {
			return -1;
		}
		if (rightPart === undefined) {
			return 1;
		}
		if (leftPart === rightPart) {
			continue;
		}
		if (typeof leftPart === 'number' && typeof rightPart === 'number') {
			return leftPart > rightPart ? 1 : -1;
		}
		if (typeof leftPart === 'number') {
			return -1;
		}
		if (typeof rightPart === 'number') {
			return 1;
		}
		return leftPart > rightPart ? 1 : -1;
	}
	return 0;
}

export function isNewerVersion(latest: string, current: string): boolean {
	return compareVersions(latest, current) === 1;
}

export function buildUpdateCheck(latestRaw: string, currentRaw: string): UpdateCheck {
	const latestVersion = parseVersion(latestRaw);
	const currentVersion = parseVersion(currentRaw);
	if (!latestVersion || !currentVersion) {
		throw new Error('La versión no tiene un formato válido.');
	}

	const latest = formatVersion(latestVersion);
	const current = formatVersion(currentVersion);
	const comparison = compareVersions(latest, current);
	const upToDate = comparison === null || comparison <= 0;
	return {
		latest,
		current,
		upToDate,
		message: upToDate
			? `Estás al día (${current})`
			: `Hay una versión nueva: ${latest}`
	};
}

export async function fetchLatestVersion(
	fetchImpl: typeof fetch,
	url = UPDATE_VERSION_URL,
	timeoutMs = 5000
): Promise<string> {
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), timeoutMs);
	try {
		const response = await fetchImpl(url, {
			headers: { Accept: 'text/plain' },
			signal: controller.signal,
			cache: 'no-store'
		});
		if (!response.ok) {
			throw new Error(`No se pudo consultar la versión publicada (${response.status}).`);
		}
		const text = (await response.text()).trim();
		if (!parseVersion(text)) {
			throw new Error('La versión publicada no tiene un formato válido.');
		}
		return text;
	} catch (reason) {
		if (reason instanceof Error && reason.name === 'AbortError') {
			throw new Error('La consulta de actualizaciones tardó demasiado.');
		}
		if (reason instanceof Error && reason.message) {
			throw reason;
		}
		throw new Error('No se pudo consultar la versión publicada.');
	} finally {
		clearTimeout(timer);
	}
}
