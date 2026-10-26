import type { MediaItem, PlaybackProgress } from './api';

export const PROGRESS_INTERVAL_MS = 15_000;
export const RESUME_END_GUARD_SECONDS = 15;

export type ProgressEvent = 'timeupdate' | 'pause' | 'ended' | 'pagehide';

export interface ProgressSnapshot {
	position: number;
	duration: number;
}

export interface ProgressDecision {
	send: boolean;
	position: number;
	duration: number;
}

// Decide si un evento del reproductor debe generar una petición de progreso.
// Las pausas, el final y la salida de la página no esperan al intervalo.
export function decideProgress(
	event: ProgressEvent,
	snapshot: ProgressSnapshot,
	lastSentAt: number | null,
	now: number
): ProgressDecision {
	const { position, duration } = snapshot;
	if (!Number.isFinite(duration) || duration <= 0) {
		return { send: false, position: 0, duration: 0 };
	}
	if (!Number.isFinite(position) || position < 0) {
		return { send: false, position: 0, duration };
	}

	const normalized = Math.min(position, duration);
	if (event === 'timeupdate') {
		if (lastSentAt !== null && now - lastSentAt < PROGRESS_INTERVAL_MS) {
			return { send: false, position: normalized, duration };
		}
		return { send: true, position: normalized, duration };
	}
	if (event === 'ended') {
		return { send: true, position: duration, duration };
	}
	return { send: true, position: normalized, duration };
}

export function progressPercent(position: number, duration: number): number {
	if (
		!Number.isFinite(position) ||
		!Number.isFinite(duration) ||
		duration <= 0 ||
		position <= 0
	) {
		return 0;
	}
	return Math.min(100, Math.max(0, Math.round((position / duration) * 100)));
}

export function formatProgressTime(seconds: number | null): string {
	if (seconds === null || !Number.isFinite(seconds) || seconds <= 0) {
		return '';
	}
	const total = Math.floor(seconds);
	const hours = Math.floor(total / 3600);
	const minutes = Math.floor((total % 3600) / 60);
	if (hours > 0) {
		return minutes > 0 ? `${hours} h ${minutes} min` : `${hours} h`;
	}
	if (minutes > 0) {
		return `${minutes} min`;
	}
	return `${total} s`;
}

export function resolveStartAt(
	progress: Pick<PlaybackProgress, 'position_seconds'> | null,
	tParam: string | null
): number {
	if (tParam !== null && tParam.trim() !== '') {
		const override = Number(tParam);
		if (Number.isFinite(override) && override >= 0) {
			return override;
		}
	}
	const position = progress?.position_seconds;
	if (position !== undefined && Number.isFinite(position) && position > 0) {
		return position;
	}
	return 0;
}

export function resumeTarget(
	startAt: number,
	duration: number,
	alreadyResumed: boolean
): number | null {
	if (alreadyResumed || !Number.isFinite(startAt) || startAt <= 0) {
		return null;
	}
	if (
		Number.isFinite(duration) &&
		duration > 0 &&
		startAt > duration - RESUME_END_GUARD_SECONDS
	) {
		return null;
	}
	return startAt;
}

export function progressEnabled(itemId: number | null | undefined): itemId is number {
	return typeof itemId === 'number' && Number.isInteger(itemId) && itemId > 0;
}

export function episodeProgressSubtitle(
	item: Pick<MediaItem, 'kind' | 'season' | 'episode'>
): string {
	if (
		item.kind === 'episode' &&
		item.season !== null &&
		item.episode !== null
	) {
		return `Serie · T${item.season} E${item.episode}`;
	}
	return '';
}
