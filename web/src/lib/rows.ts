import type { Credits } from './api';

export interface RowScrollState {
	canPrev: boolean;
	canNext: boolean;
	prevOffset: number;
	nextOffset: number;
	overflow: boolean;
}

const EDGE_TOLERANCE = 1;

export function rowScrollState(
	scrollLeft: number,
	clientWidth: number,
	scrollWidth: number
): RowScrollState {
	const left = Number.isFinite(scrollLeft) ? Math.max(0, scrollLeft) : 0;
	const width = Number.isFinite(clientWidth) ? Math.max(0, clientWidth) : 0;
	const total = Number.isFinite(scrollWidth) ? Math.max(0, scrollWidth) : 0;
	const maxScroll = Math.max(0, total - width);
	const overflow = maxScroll > EDGE_TOLERANCE;
	const page = width;
	return {
		canPrev: overflow && left > EDGE_TOLERANCE,
		canNext: overflow && left < maxScroll - EDGE_TOLERANCE,
		prevOffset: Math.max(0, Math.min(maxScroll, left - page)),
		nextOffset: Math.max(0, Math.min(maxScroll, left + page)),
		overflow
	};
}

export interface CreditRow {
	key: string;
	name: string;
	photo: string | null;
	subtitle: string;
}

const CREW_PRIORITY: Record<string, number> = {
	Director: 0,
	Writer: 1,
	Screenplay: 1,
	Novel: 1,
	Story: 1,
	Producer: 2,
	'Executive Producer': 2,
	'Original Music Composer': 3,
	'Director of Photography': 4,
	Editor: 5,
	Creator: 6
};

export function crewPriority(job: string): number {
	return CREW_PRIORITY[job.trim()] ?? 7;
}

export function creditRows(
	credits: Credits | null | undefined,
	cast: string[],
	directors: string[]
): CreditRow[] {
	const result: CreditRow[] = [];
	const castCredits = credits?.cast ?? [];
	if (castCredits.length > 0) {
		for (const [index, member] of castCredits.entries()) {
			if (member.name.trim()) {
				result.push({
					key: `cast-${index}-${member.name}`,
					name: member.name,
					photo: member.photo_url.trim() || null,
					subtitle: member.character.trim() ? `como ${member.character}` : ''
				});
			}
		}
	} else {
		for (const [index, name] of cast.entries()) {
			if (name.trim()) {
				result.push({ key: `cast-name-${index}-${name}`, name, photo: null, subtitle: '' });
			}
		}
	}

	const crewCredits = credits?.crew ?? [];
	if (crewCredits.length > 0) {
		const ordered = crewCredits
			.map((member, index) => ({ member, index, priority: crewPriority(member.job) }))
			.filter((entry) => entry.member.name.trim())
			.sort((a, b) => a.priority - b.priority || a.index - b.index);
		for (const entry of ordered) {
			const member = entry.member;
			result.push({
				key: `crew-${entry.index}-${member.name}`,
				name: member.name,
				photo: member.photo_url.trim() || null,
				subtitle: jobLabel(member.job)
			});
		}
	} else {
		for (const [index, name] of directors.entries()) {
			if (name.trim()) {
				result.push({
					key: `director-${index}-${name}`,
					name,
					photo: null,
					subtitle: 'Dirección'
				});
			}
		}
	}
	return result;
}

export function jobLabel(job: string): string {
	switch (job.trim()) {
		case 'Director':
			return 'Dirección';
		case 'Writer':
		case 'Screenplay':
		case 'Novel':
		case 'Story':
			return 'Guion';
		case 'Producer':
		case 'Executive Producer':
			return 'Producción';
		case 'Original Music Composer':
			return 'Música';
		case 'Director of Photography':
			return 'Fotografía';
		case 'Editor':
			return 'Montaje';
		case 'Creator':
			return 'Creación';
		default:
			return job.trim();
	}
}

export function personInitials(name: string): string {
	const words = name
		.trim()
		.split(/\s+/)
		.filter(Boolean);
	if (words.length === 0) {
		return '';
	}
	const first = firstLetter(words[0]);
	if (words.length === 1) {
		return first;
	}
	return first + firstLetter(words[words.length - 1]);
}

function firstLetter(word: string): string {
	const normalized = word.normalize('NFD').replace(/[\u0300-\u036f]/g, '');
	return (normalized[0] ?? word[0] ?? '').toUpperCase();
}
