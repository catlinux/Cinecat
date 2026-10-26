import type { UpdateInfo, UpdateStatus } from './api';

export const UPDATE_INSTALL_COMMAND = './scripts/instalar-actualizador.sh';
export const UPDATE_MANUAL_COMMAND = './scripts/actualizar.sh';
export const UPDATE_POLL_INTERVAL = 3000;
export const UPDATE_POLL_MAX_INTERVAL = 10000;
export const UPDATE_TRACKING_LIMIT = 30 * 60 * 1000;

export interface ParsedVersion {
	major: number;
	minor: number;
	patch: number;
	prerelease: Array<string | number>;
}

export interface UpdateActionState {
	updateAction?: boolean;
	success?: boolean;
	message?: string;
	updateInfo?: UpdateInfo | null;
	appliedTag?: string | null;
	installHelp?: boolean;
}

export type UpdateOutcome = 'ok' | 'failed' | 'rolled_back' | null;

const PHASE_LABELS: Record<string, string> = {
	Validando: 'Validando la petición',
	'Comprobando espacio': 'Comprobando espacio libre',
	'Copia de la base de datos': 'Guardando una copia de la base de datos',
	'Descargando la versión': 'Descargando la versión',
	Compilando: 'Compilando la versión nueva',
	Reiniciando: 'Reiniciando el servidor',
	'Comprobando que arranca': 'Comprobando que arranca',
	Terminado: 'Terminado',
	Revertido: 'Revertido'
};

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

export function versionMatches(current: string, tag: string): boolean {
	const comparison = compareVersions(current, tag);
	if (comparison !== null) {
		return comparison === 0;
	}
	return current.trim().replace(/^v/, '') === tag.trim().replace(/^v/, '');
}

export function isUpdateInProgress(status: UpdateStatus | null | undefined): boolean {
	return Boolean(status && status.result === '' && status.phase.trim() !== '');
}

export function phaseLabel(phase: string | null | undefined): string {
	const text = String(phase ?? '').trim();
	return PHASE_LABELS[text] ?? (text || 'Actualización en curso');
}

export function updatedTag(
	info: UpdateInfo | null | undefined,
	expectedTag: string | null = null
): string | null {
	const status = info?.status;
	if (!status || status.result !== 'ok' || !status.tag) {
		return null;
	}
	if (expectedTag && !versionMatches(status.tag, expectedTag)) {
		return null;
	}
	return versionMatches(info.current, status.tag) ? status.tag : null;
}

export function updateOutcome(
	info: UpdateInfo | null | undefined,
	expectedTag: string | null = null
): UpdateOutcome {
	const status = info?.status;
	const result = status?.result;
	if (expectedTag && (!status?.tag || !versionMatches(status.tag, expectedTag))) {
		return null;
	}
	if (result === 'ok') {
		return updatedTag(info, expectedTag) ? 'ok' : null;
	}
	if (result === 'failed' || result === 'rolled_back') {
		return result;
	}
	return null;
}

export function elapsedSeconds(startedAt: string | null | undefined, now = Date.now()): number {
	if (!startedAt) {
		return 0;
	}
	const started = Date.parse(startedAt);
	if (Number.isNaN(started)) {
		return 0;
	}
	return Math.max(0, Math.floor((now - started) / 1000));
}

export function formatElapsed(totalSeconds: number): string {
	const seconds = Math.max(0, Math.floor(totalSeconds));
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	const remainder = seconds % 60;
	if (hours > 0) {
		return `${hours}:${String(minutes).padStart(2, '0')}:${String(remainder).padStart(2, '0')}`;
	}
	return `${minutes}:${String(remainder).padStart(2, '0')}`;
}

export function pollDelay(errorCount: number): number {
	const failures = Number.isFinite(errorCount) ? Math.max(0, Math.floor(errorCount)) : 0;
	if (failures === 0) {
		return UPDATE_POLL_INTERVAL;
	}
	return Math.min(UPDATE_POLL_INTERVAL * 2 ** failures, UPDATE_POLL_MAX_INTERVAL);
}

export function trackingExpired(
	startedAt: string | null | undefined,
	now = Date.now(),
	limit = UPDATE_TRACKING_LIMIT
): boolean {
	if (!startedAt) {
		return false;
	}
	const started = Date.parse(startedAt);
	if (Number.isNaN(started)) {
		return false;
	}
	return now - started >= limit;
}
