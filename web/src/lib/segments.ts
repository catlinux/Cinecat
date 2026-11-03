import type { ItemSegment, SegmentKind } from './api';

export const SEGMENT_KINDS: SegmentKind[] = ['intro', 'credits'];

export const SEGMENT_LABELS: Record<SegmentKind, string> = {
	intro: 'Intro',
	credits: 'Créditos'
};

export interface SegmentTimeDraft {
	start: string;
	end: string;
}

export type SegmentDraftMap = Record<SegmentKind, SegmentTimeDraft>;

export interface ParsedSegmentForm {
	segments: ItemSegment[];
	values: SegmentDraftMap;
	error: string | null;
}

export function emptySegmentDrafts(): SegmentDraftMap {
	return {
		intro: { start: '', end: '' },
		credits: { start: '', end: '' }
	};
}

export function formatSegmentTime(seconds: number): string {
	if (!Number.isFinite(seconds) || seconds < 0) {
		return '';
	}
	const total = Math.floor(seconds);
	const hours = Math.floor(total / 3600);
	const minutes = Math.floor((total % 3600) / 60);
	const rest = total % 60;
	if (hours > 0) {
		return `${hours}:${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`;
	}
	return `${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`;
}

export function parseSegmentTime(value: string): number | null {
	const text = value.trim();
	if (!text) {
		return null;
	}
	const parts = text.split(':');
	if (parts.length !== 2 && parts.length !== 3) {
		return null;
	}
	const [hoursText, minutesText, secondsText] =
		parts.length === 3 ? parts : ['0', parts[0], parts[1]];
	const minutesPattern = parts.length === 3 ? /^\d{2}$/ : /^\d{1,2}$/;
	if (
		!/^\d+$/.test(hoursText) ||
		!minutesPattern.test(minutesText) ||
		!/^\d{2}$/.test(secondsText)
	) {
		return null;
	}
	const hours = Number(hoursText);
	const minutes = Number(minutesText);
	const seconds = Number(secondsText);
	if (minutes > 59 || seconds > 59) {
		return null;
	}
	return hours * 3600 + minutes * 60 + seconds;
}

export function segmentDrafts(manual: ItemSegment[] | null | undefined): SegmentDraftMap {
	const result = emptySegmentDrafts();
	for (const segment of manual ?? []) {
		if (segment.kind === 'intro' || segment.kind === 'credits') {
			result[segment.kind] = {
				start: formatSegmentTime(segment.start),
				end: formatSegmentTime(segment.end)
			};
		}
	}
	return result;
}

export function parseSegmentForm(values: SegmentDraftMap): ParsedSegmentForm {
	const segments: ItemSegment[] = [];
	for (const kind of SEGMENT_KINDS) {
		const startText = values[kind].start.trim();
		const endText = values[kind].end.trim();
		// Dejar una fila vacía es la forma de borrar ese segmento manual.
		if (!startText || !endText) {
			continue;
		}
		const start = parseSegmentTime(startText);
		const end = parseSegmentTime(endText);
		if (start === null || end === null) {
			return {
				segments: [],
				values,
				error: `El tiempo de ${SEGMENT_LABELS[kind]} no es válido. Usa mm:ss o h:mm:ss.`
			};
		}
		if (end <= start) {
			return {
				segments: [],
				values,
				error: `El fin de ${SEGMENT_LABELS[kind]} debe ser posterior al inicio.`
			};
		}
		segments.push({ kind, start, end });
	}
	return { segments, values, error: null };
}
