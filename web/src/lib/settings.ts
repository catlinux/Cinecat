import type {
	Setting,
	SettingsChanges,
	SettingsResponse,
	SettingsSection,
	SettingsValues
} from './api';
import type { ThemePreference } from './theme';

export type NavLayout = 'side' | 'top';
export type Accent = 'granate' | 'naranja' | 'dorado';

export interface PublicUI {
	server_name: string;
	tagline: string;
	portal_title: string;
	portal_intro: string;
	nav_layout: NavLayout;
	show_collections: boolean;
	show_series: boolean;
	theme_default: ThemePreference;
	accent: Accent;
	grid_columns: number;
	grid_rows: number;
	row_columns: number;
}

export const DEFAULT_PUBLIC_UI: PublicUI = {
	server_name: 'Cinecat',
	tagline: 'Sesión continua',
	portal_title: 'Tu cine en casa',
	portal_intro:
		'Una biblioteca personal con olor a palomitas, carteles pintados a mano y la sesión de siempre esperándote.',
	nav_layout: 'side',
	show_collections: true,
	show_series: true,
	theme_default: 'system',
	accent: 'granate',
	grid_columns: 6,
	grid_rows: 8,
	row_columns: 6
};

function publicText(value: unknown, fallback: string, allowEmpty = false): string {
	if (typeof value !== 'string') {
		return fallback;
	}
	const text = value.trim();
	return text === '' && !allowEmpty ? fallback : text;
}

function publicFlag(value: unknown, fallback: boolean): boolean {
	return typeof value === 'boolean' ? value : fallback;
}

function publicInt(value: unknown, fallback: number, minimum: number, maximum: number): number {
	if (typeof value !== 'number' || !Number.isInteger(value)) {
		return fallback;
	}
	return value >= minimum && value <= maximum ? value : fallback;
}

export function listPageSize(ui?: Partial<PublicUI> | null): number {
	const columns = publicInt(
		ui?.grid_columns,
		DEFAULT_PUBLIC_UI.grid_columns,
		4,
		8
	);
	const rows = publicInt(ui?.grid_rows, DEFAULT_PUBLIC_UI.grid_rows, 2, 12);
	return columns * rows;
}

export function parsePublicUI(raw: Record<string, unknown> | null | undefined): PublicUI {
	const source = raw ?? {};
	const navLayout = source['ui.nav_layout'];
	const themeDefault = source['ui.theme_default'];
	const accent = source['ui.accent'];
	return {
		server_name: publicText(source['ui.server_name'], DEFAULT_PUBLIC_UI.server_name),
		tagline: publicText(source['ui.tagline'], DEFAULT_PUBLIC_UI.tagline, true),
		portal_title: publicText(source['ui.portal_title'], DEFAULT_PUBLIC_UI.portal_title),
		portal_intro: publicText(source['ui.portal_intro'], DEFAULT_PUBLIC_UI.portal_intro, true),
		nav_layout: navLayout === 'top' ? 'top' : 'side',
		show_collections: publicFlag(
			source['ui.show_collections'],
			DEFAULT_PUBLIC_UI.show_collections
		),
		show_series: publicFlag(source['ui.show_series'], DEFAULT_PUBLIC_UI.show_series),
		theme_default:
			themeDefault === 'light' || themeDefault === 'dark' || themeDefault === 'system'
				? themeDefault
				: DEFAULT_PUBLIC_UI.theme_default,
		accent:
			accent === 'naranja' || accent === 'dorado' || accent === 'granate'
				? accent
				: DEFAULT_PUBLIC_UI.accent,
		grid_columns: publicInt(source['ui.grid_columns'], DEFAULT_PUBLIC_UI.grid_columns, 4, 8),
		grid_rows: publicInt(source['ui.grid_rows'], DEFAULT_PUBLIC_UI.grid_rows, 2, 12),
		row_columns: publicInt(source['ui.row_columns'], DEFAULT_PUBLIC_UI.row_columns, 3, 8)
	};
}

export const settingsSectionKeys = [
	'scan',
	'metadata',
	'ui',
	'playback',
	'library',
	'system'
] as const;

export type SettingsSectionKey = (typeof settingsSectionKeys)[number];

export interface SettingsFormState {
	success?: boolean;
	message?: string;
	fields?: Record<string, string>;
	values?: SettingsValues;
}

export interface CollectedSettingsForm {
	changes: SettingsChanges;
	fields: Record<string, string>;
	values: SettingsValues;
}

export interface ParsedSettingValue {
	value?: unknown;
	error?: string;
}

export function isSettingsSection(value: string): value is SettingsSectionKey {
	return (settingsSectionKeys as readonly string[]).includes(value);
}

export function findSettingsSection(
	response: SettingsResponse,
	key: string
): SettingsSection | null {
	return response.sections.find((section) => section.key === key) ?? null;
}

export function settingControlId(key: string): string {
	return `setting-${key.replace(/[^a-zA-Z0-9_-]+/g, '-')}`;
}

export function settingDescriptionId(key: string): string {
	return `${settingControlId(key)}-description`;
}

export function settingErrorId(key: string): string {
	return `${settingControlId(key)}-error`;
}

export function isNumericEnum(setting: Setting): boolean {
	return (
		setting.type === 'enum' &&
		setting.options.length > 0 &&
		setting.options.every((option) => /^-?\d+$/.test(option.value.trim()))
	);
}

function formatDecimal(value: number, maximumFractionDigits = 4): string {
	if (!Number.isFinite(value)) {
		return '—';
	}
	return new Intl.NumberFormat('es-ES', {
		maximumFractionDigits,
		useGrouping: false
	}).format(value);
}

export function formatDurationText(value: unknown): string {
	const text = String(value ?? '').trim();
	if (text === '') {
		return '—';
	}
	if (text === '0' || text === '0s' || text === '0m' || text === '0h') {
		return '0';
	}

	const parts: string[] = [];
	const expression = /(\d+(?:[.,]\d+)?)(h|m|s)/g;
	let consumed = '';
	for (const match of text.matchAll(expression)) {
		const amount = Number(match[1].replace(',', '.'));
		consumed += match[0];
		if (!Number.isFinite(amount) || amount === 0) {
			continue;
		}
		const unit = match[2] === 'h' ? 'h' : match[2] === 'm' ? 'min' : 's';
		parts.push(`${formatDecimal(amount)} ${unit}`);
	}
	if (parts.length === 0 || consumed !== text) {
		return text;
	}
	return parts.join(' ');
}

export function formatSettingDefault(setting: Setting): string {
	const value = setting.default;
	if (value === null || value === undefined) {
		return '—';
	}
	switch (setting.type) {
		case 'bool':
			return value ? 'Sí' : 'No';
		case 'duration':
			return formatDurationText(value);
		case 'float':
			return formatDecimal(Number(value));
		case 'enum': {
			const option = setting.options.find((item) => item.value === String(value));
			return option?.label ?? String(value);
		}
		default:
			return String(value) || '—';
	}
}

export function formatSettingInputValue(setting: Setting, value: unknown): string | boolean {
	if (setting.type === 'bool') {
		return Boolean(value);
	}
	if (value === null || value === undefined) {
		return '';
	}
	return String(value);
}

export function parseSettingValue(
	setting: Setting,
	raw: FormDataEntryValue | boolean | null | undefined
): ParsedSettingValue {
	if (setting.type === 'bool') {
		const checked = raw === true || raw === '1' || raw === 'on' || raw === 'true';
		return { value: checked };
	}

	const text = String(raw ?? '').trim();
	switch (setting.type) {
		case 'int':
		case 'float': {
			if (text === '') {
				return { error: setting.type === 'int' ? 'Escribe un número entero.' : 'Escribe un número.' };
			}
			const value = Number(text.replace(',', '.'));
			if (!Number.isFinite(value) || (setting.type === 'int' && !Number.isInteger(value))) {
				return {
					error: setting.type === 'int' ? 'Escribe un número entero.' : 'Escribe un número.'
				};
			}
			const rangeError = validateRange(setting, value);
			if (rangeError) {
				return { error: rangeError };
			}
			return { value };
		}
		case 'string': {
			const length = [...text].length;
			if (setting.min > 0 && length < setting.min) {
				return { error: `El texto debe tener al menos ${setting.min} caracteres.` };
			}
			if (setting.max > 0 && length > setting.max) {
				return { error: `El texto no puede superar los ${setting.max} caracteres.` };
			}
			return { value: text };
		}
		case 'duration': {
			if (text === '') {
				return { error: 'Escribe una duración (por ejemplo, 6h o 30m).' };
			}
			if (!/^(?:0|(?:\d+(?:[.,]\d+)?(?:h|m|s))+)$/.test(text)) {
				return { error: 'Duración no válida. Usa, por ejemplo, 6h o 30m.' };
			}
			return { value: text.replace(',', '.') };
		}
		case 'enum': {
			const option = setting.options.find((item) => item.value === text);
			if (!option) {
				return { error: 'Elige una opción válida.' };
			}
			return { value: isNumericEnum(setting) ? Number(text) : text };
		}
		default:
			return { value: text };
	}
}

function validateRange(setting: Setting, value: number): string | null {
	if (setting.min !== 0 && value < setting.min) {
		return `El valor debe ser igual o mayor que ${formatDecimal(setting.min)}.`;
	}
	if (setting.max !== 0 && value > setting.max) {
		return `El valor debe ser igual o menor que ${formatDecimal(setting.max)}.`;
	}
	return null;
}

function valuesEqual(left: unknown, right: unknown): boolean {
	return Object.is(left, right);
}

export function collectSettingsForm(
	settings: Setting[],
	formData: FormData
): CollectedSettingsForm {
	const changes: SettingsChanges = {};
	const fields: Record<string, string> = {};
	const values: SettingsValues = {};

	for (const setting of settings) {
		const raw = formData.get(setting.key);

		if (setting.type === 'bool') {
			const parsed = parseSettingValue(setting, raw);
			if (parsed.error || parsed.value === undefined) {
				fields[setting.key] = parsed.error ?? 'Valor no válido.';
				values[setting.key] = false;
				continue;
			}
			const value = Boolean(parsed.value);
			values[setting.key] = value;
			if (!valuesEqual(value, setting.value)) {
				changes[setting.key] = value;
			}
			continue;
		}

		if (raw === null) {
			values[setting.key] = formatSettingInputValue(setting, setting.value);
			continue;
		}

		const parsed = parseSettingValue(setting, raw);
		values[setting.key] = String(raw);
		if (parsed.error || parsed.value === undefined) {
			fields[setting.key] = parsed.error ?? 'Valor no válido.';
			continue;
		}
		if (!valuesEqual(parsed.value, setting.value)) {
			changes[setting.key] = parsed.value;
		}
	}

	return { changes, fields, values };
}
