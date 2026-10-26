export const PLAYER_SPEEDS = [0.5, 0.75, 1, 1.25, 1.5, 2] as const;
export const CONTROLS_HIDE_MS = 3000;
export const SEEK_STEP_SECONDS = 10;
export const SEEK_STEP_LONG_SECONDS = 30;
export const FRAME_STEP_SECONDS = 1 / 24;

export type PlayerShortcut =
	| { type: 'toggle' }
	| { type: 'seek'; value: number }
	| { type: 'volume'; value: number }
	| { type: 'mute' }
	| { type: 'fullscreen' }
	| { type: 'subtitles' }
	| { type: 'frame'; delta: number }
	| { type: 'percent'; value: number }
	| { type: 'escape' };

export interface ShortcutContext {
	key: string;
	ctrlKey?: boolean;
	metaKey?: boolean;
	altKey?: boolean;
	shiftKey?: boolean;
	paused: boolean;
	duration: number;
	currentTime: number;
	volume: number;
}

export function formatPlayerTime(seconds: number): string {
	if (!Number.isFinite(seconds) || seconds < 0) {
		return '0:00:00';
	}
	const total = Math.floor(seconds);
	const hours = Math.floor(total / 3600);
	const minutes = Math.floor((total % 3600) / 60);
	const rest = total % 60;
	return `${hours}:${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`;
}

export function clampSeekTime(current: number, delta: number, duration: number): number {
	const max = Number.isFinite(duration) && duration > 0 ? duration : 0;
	return Math.min(max, Math.max(0, (Number.isFinite(current) ? current : 0) + delta));
}

export function pointerPosition(
	clientX: number,
	left: number,
	width: number,
	duration: number
): number {
	if (!Number.isFinite(duration) || duration <= 0 || !Number.isFinite(width) || width <= 0) {
		return 0;
	}
	const ratio = Math.min(1, Math.max(0, (clientX - left) / width));
	return ratio * duration;
}

export function cycleSpeed(current: number, direction = 1): number {
	const index = PLAYER_SPEEDS.indexOf(current as (typeof PLAYER_SPEEDS)[number]);
	if (index === -1) {
		return 1;
	}
	const next = (index + direction + PLAYER_SPEEDS.length) % PLAYER_SPEEDS.length;
	return PLAYER_SPEEDS[next];
}

export function formatSpeed(speed: number): string {
	return `${String(speed).replace('.', ',')}×`;
}

export function controlsShouldHide(
	playing: boolean,
	menuOpen: boolean,
	controlFocused: boolean,
	lastActivity: number,
	now: number
): boolean {
	return playing && !menuOpen && !controlFocused && now - lastActivity > CONTROLS_HIDE_MS;
}

export function shortcutAction(context: ShortcutContext): PlayerShortcut | null {
	if (context.ctrlKey || context.metaKey || context.altKey) {
		return null;
	}
	const key = context.key;
	const seek = context.shiftKey ? SEEK_STEP_LONG_SECONDS : SEEK_STEP_SECONDS;
	switch (key) {
		case ' ':
		case 'k':
		case 'K':
			return { type: 'toggle' };
		case 'ArrowLeft':
		case 'j':
		case 'J':
			return { type: 'seek', value: clampSeekTime(context.currentTime, -seek, context.duration) };
		case 'ArrowRight':
		case 'l':
		case 'L':
			return { type: 'seek', value: clampSeekTime(context.currentTime, seek, context.duration) };
		case 'ArrowUp':
			return { type: 'volume', value: Math.min(1, context.volume + 0.05) };
		case 'ArrowDown':
			return { type: 'volume', value: Math.max(0, context.volume - 0.05) };
		case 'm':
		case 'M':
			return { type: 'mute' };
		case 'f':
		case 'F':
			return { type: 'fullscreen' };
		case 'c':
		case 'C':
			return { type: 'subtitles' };
		case ',':
			return context.paused ? { type: 'frame', delta: -FRAME_STEP_SECONDS } : null;
		case '.':
			return context.paused ? { type: 'frame', delta: FRAME_STEP_SECONDS } : null;
		case 'Escape':
			return { type: 'escape' };
		default:
			if (key >= '0' && key <= '9' && Number.isFinite(context.duration) && context.duration > 0) {
				return { type: 'percent', value: (Number(key) / 10) * context.duration };
			}
			return null;
	}
}

export function absoluteSubtitleURL(base: string, path: string): string {
	if (/^https?:\/\//i.test(path)) {
		return path;
	}
	try {
		return new URL(path, base).toString();
	} catch {
		return path;
	}
}

export function countdownLabel(seconds: number): string {
	return `Siguiente episodio en ${Math.max(0, Math.ceil(seconds))}…`;
}
