import { describe, expect, it } from 'vitest';
import type { Setting } from './api';
import {
	collectSettingsForm,
	DEFAULT_PORTAL_ROWS,
	DEFAULT_PUBLIC_UI,
	findSettingsSection,
	formatDurationText,
	formatSettingDefault,
	isNumericEnum,
	listPageSize,
	parsePortalRows,
	parsePublicUI,
	parseSettingValue
} from './settings';

function setting(overrides: Partial<Setting> & Pick<Setting, 'key' | 'type'>): Setting {
	return {
		label: overrides.key,
		description: 'Descripción',
		options: [],
		min: 0,
		max: 0,
		default: null,
		value: null,
		modified: false,
		...overrides
	};
}

const gridColumns = setting({
	key: 'ui.grid_columns',
	type: 'int',
	default: 6,
	value: 6,
	min: 4,
	max: 8
});
const maxHeight = setting({
	key: 'playback.max_height',
	type: 'enum',
	default: 1080,
	value: 1080,
	options: [
		{ value: '480', label: '480p' },
		{ value: '720', label: '720p' },
		{ value: '1080', label: '1080p' }
	],
	min: 480,
	max: 1080
});
const language = setting({
	key: 'metadata.language',
	type: 'enum',
	default: 'es-ES',
	value: 'es-ES',
	options: [
		{ value: 'es-ES', label: 'Español (España)' },
		{ value: 'en-US', label: 'Inglés' }
	]
});
const showSeries = setting({
	key: 'ui.show_series',
	type: 'bool',
	default: true,
	value: true
});
const serverName = setting({
	key: 'ui.server_name',
	type: 'string',
	default: 'Cinecat',
	value: 'Cinecat',
	min: 1,
	max: 40
});
const interval = setting({
	key: 'scan.interval',
	type: 'duration',
	default: '6h',
	value: '6h',
	min: 60
});
const crf = setting({
	key: 'playback.crf',
	type: 'int',
	default: 20,
	value: 20,
	min: 18,
	max: 28
});
const threshold = setting({
	key: 'metadata.match_threshold',
	type: 'float',
	default: 0.75,
	value: 0.75,
	min: 0.6,
	max: 0.95
});

describe('ajustes públicos de la interfaz', () => {
	it('usa los mismos valores por defecto que el servidor', () => {
		expect(DEFAULT_PUBLIC_UI).toMatchObject({
			server_name: 'Cinecat',
			tagline: 'Sesión continua',
			portal_title: 'Tu cine en casa',
			nav_layout: 'side',
			show_collections: true,
			show_series: true,
			theme_default: 'system',
			accent: 'granate',
			grid_columns: 6,
			grid_rows: 8,
			row_columns: 6
		});
	});

	it('lee los valores válidos de la API', () => {
		const ui = parsePublicUI({
			'ui.server_name': '  Mi cine  ',
			'ui.tagline': '  Siempre en sesión  ',
			'ui.portal_title': '  Mi cartelera  ',
			'ui.portal_intro': '  Una sala luminosa.  ',
			'ui.nav_layout': 'top',
			'ui.show_collections': false,
			'ui.show_series': false,
			'ui.theme_default': 'dark',
			'ui.accent': 'dorado',
			'ui.grid_columns': 5,
			'ui.grid_rows': 10,
			'ui.row_columns': 4,
			'ui.portal_rows': JSON.stringify([
				{ id: 'favorites', visible: false, title: '  Mi selección  ', kicker: 'Lo mejor' }
			])
		});

		expect(ui).toEqual({
			server_name: 'Mi cine',
			tagline: 'Siempre en sesión',
			portal_title: 'Mi cartelera',
			portal_intro: 'Una sala luminosa.',
			nav_layout: 'top',
			show_collections: false,
			show_series: false,
			theme_default: 'dark',
			accent: 'dorado',
			grid_columns: 5,
			grid_rows: 10,
			row_columns: 4,
			portal_rows: parsePortalRows([
				{ id: 'favorites', visible: false, title: 'Mi selección', kicker: 'Lo mejor' }
			])
		});
	});

	it('conserva vacíos el eslogan y la introducción', () => {
		const ui = parsePublicUI({
			'ui.tagline': '   ',
			'ui.portal_intro': ''
		});

		expect(ui.tagline).toBe('');
		expect(ui.portal_intro).toBe('');
	});

	it('cae en los valores por defecto ante una API incompleta o inválida', () => {
		expect(parsePublicUI(null)).toEqual(DEFAULT_PUBLIC_UI);
		expect(
			parsePublicUI({
				'ui.server_name': 42,
				'ui.portal_title': '',
				'ui.nav_layout': 'left',
				'ui.show_collections': 'no',
				'ui.show_series': null,
				'ui.theme_default': 'sepia',
				'ui.accent': 'verde',
				'ui.grid_columns': 3,
				'ui.grid_rows': 13,
				'ui.row_columns': 9
			})
		).toEqual(DEFAULT_PUBLIC_UI);
	});

	it('calcula los resultados por página como columnas × filas', () => {
		expect(listPageSize({ grid_columns: 6, grid_rows: 8 })).toBe(48);
		expect(listPageSize({ grid_columns: 5, grid_rows: 10 })).toBe(50);
		expect(listPageSize({ grid_columns: 3, grid_rows: 99 })).toBe(48);
		expect(listPageSize(null)).toBe(48);
	});
});

describe('filas del portal', () => {
	it('usa el orden y los textos por defecto', () => {
		expect(DEFAULT_PORTAL_ROWS.map((row) => row.id)).toEqual([
			'recent_movies',
			'continue_movies',
			'watchlist_movies',
			'watched_movies',
			'recent_series',
			'continue_series',
			'next_up',
			'watchlist_series',
			'favorites'
		]);
		expect(DEFAULT_PORTAL_ROWS[3]).toMatchObject({
			title: 'Vistas hace poco',
			kicker: 'Tu historial'
		});
	});

	it('normaliza desconocidos, duplicados, huecos y títulos vacíos', () => {
		const rows = parsePortalRows(
			JSON.stringify([
				{ id: 'favorites', visible: false, title: '  Mi selección  ', kicker: '  Lo mejor  ' },
				{ id: 'desconocida', visible: true, title: 'No', kicker: 'No' },
				{ id: 'recent_movies', visible: true, title: '   ', kicker: '' },
				{ id: 'favorites', visible: true, title: 'Duplicada', kicker: 'Duplicada' }
			])
		);
		expect(rows).toHaveLength(DEFAULT_PORTAL_ROWS.length);
		expect(rows[0]).toEqual({
			id: 'favorites',
			visible: false,
			title: 'Mi selección',
			kicker: 'Lo mejor'
		});
		expect(rows[1]).toEqual({
			id: 'recent_movies',
			visible: true,
			title: 'Películas recién añadidas',
			kicker: ''
		});
		expect(rows[2].id).toBe('continue_movies');
		expect(rows.some((row) => row.id === 'desconocida')).toBe(false);
		expect(rows.some((row) => row.title === 'Duplicada')).toBe(false);
	});

	it('vuelve a los valores por defecto con JSON inválido y tolera tipos incorrectos', () => {
		expect(parsePortalRows('{')).toEqual(DEFAULT_PORTAL_ROWS);
		expect(parsePortalRows({})).toEqual(DEFAULT_PORTAL_ROWS);
		const rows = parsePortalRows([
			{ id: 'favorites', visible: 'sí', title: 3, kicker: null }
		]);
		expect(rows[0]).toEqual({
			id: 'favorites',
			visible: true,
			title: 'Tus favoritos',
			kicker: 'Tu selección'
		});
		expect(rows.slice(1)).toEqual(DEFAULT_PORTAL_ROWS.slice(0, 8));
	});

	it('completa las filas que faltan al final', () => {
		const rows = parsePortalRows('[{"id":"next_up","visible":false}]');
		expect(rows[0]).toMatchObject({ id: 'next_up', visible: false, title: 'Siguiente episodio' });
		expect(rows.slice(1).map((row) => row.id)).toEqual([
			'recent_movies',
			'continue_movies',
			'watchlist_movies',
			'watched_movies',
			'recent_series',
			'continue_series',
			'watchlist_series',
			'favorites'
		]);
	});
});

describe('formato de ajustes', () => {
	it('distingue los enum numéricos', () => {
		expect(isNumericEnum(maxHeight)).toBe(true);
		expect(isNumericEnum(language)).toBe(false);
	});

	it('formatea duraciones para el valor por defecto', () => {
		expect(formatDurationText('6h')).toBe('6 h');
		expect(formatDurationText('30m')).toBe('30 min');
		expect(formatDurationText('1h30m')).toBe('1 h 30 min');
		expect(formatDurationText('0')).toBe('0');
		expect(formatDurationText('')).toBe('—');
	});

	it('formatea el defecto según el tipo', () => {
		expect(formatSettingDefault(interval)).toBe('6 h');
		expect(formatSettingDefault(showSeries)).toBe('Sí');
		expect(formatSettingDefault(gridColumns)).toBe('6');
		expect(formatSettingDefault(threshold)).toBe('0,75');
		expect(formatSettingDefault(language)).toBe('Español (España)');
	});
});

describe('conversión de valores del formulario', () => {
	it('convierte cada tipo y aplica límites', () => {
		expect(parseSettingValue(showSeries, null)).toEqual({ value: false });
		expect(parseSettingValue(showSeries, '1')).toEqual({ value: true });
		expect(parseSettingValue(crf, '22')).toEqual({ value: 22 });
		expect(parseSettingValue(crf, '29').error).toContain('menor');
		expect(parseSettingValue(threshold, '0,8')).toEqual({ value: 0.8 });
		expect(parseSettingValue(serverName, '  Mi cine  ')).toEqual({ value: 'Mi cine' });
		expect(parseSettingValue(interval, '30m')).toEqual({ value: '30m' });
		expect(parseSettingValue(interval, '0')).toEqual({ value: '0' });
		expect(parseSettingValue(gridColumns, '7')).toEqual({ value: 7 });
		expect(parseSettingValue(language, 'en-US')).toEqual({ value: 'en-US' });
	});

	it('devuelve errores de tipo, rango y opción', () => {
		expect(parseSettingValue(crf, '2.5').error).toContain('entero');
		expect(parseSettingValue(crf, '17').error).toContain('mayor');
		expect(parseSettingValue(serverName, '').error).toContain('al menos');
		expect(parseSettingValue(interval, 'mañana').error).toContain('Duración');
		expect(parseSettingValue(gridColumns, '9').error).toContain('menor');
		expect(parseSettingValue(language, 'fr-FR').error).toContain('opción');
	});
});

describe('recogida del formulario', () => {
	it('envía solo los campos modificados y conserva el tipo', () => {
		const form = new FormData();
		form.set('ui.grid_columns', '4');
		form.set('metadata.language', 'es-ES');
		form.set('ui.server_name', 'Cinecat');
		form.set('scan.interval', '30m');
		form.set('playback.crf', '20');
		form.set('metadata.match_threshold', '0.75');

		const collected = collectSettingsForm(
			[gridColumns, language, serverName, interval, crf, threshold, showSeries],
			form
		);

		expect(collected.changes).toEqual({
			'ui.grid_columns': 4,
			'scan.interval': '30m',
			'ui.show_series': false
		});
		expect(collected.fields).toEqual({});
		expect(collected.values['ui.grid_columns']).toBe('4');
		expect(collected.values['ui.show_series']).toBe(false);
	});

	it('no envía nada si los valores no cambian', () => {
		const form = new FormData();
		form.set('ui.grid_columns', '6');
		form.set('metadata.language', 'es-ES');
		form.set('ui.server_name', 'Cinecat');
		form.set('scan.interval', '6h');
		form.set('playback.crf', '20');
		form.set('metadata.match_threshold', '0.75');
		form.set('ui.show_series', '1');

		const collected = collectSettingsForm(
			[gridColumns, language, serverName, interval, crf, threshold, showSeries],
			form
		);

		expect(collected.changes).toEqual({});
		expect(collected.fields).toEqual({});
	});

	it('conserva los valores escritos y los errores por campo', () => {
		const form = new FormData();
		form.set('playback.crf', '99');
		form.set('ui.server_name', '');
		form.set('scan.interval', '30m');

		const collected = collectSettingsForm([crf, serverName, interval], form);

		expect(collected.fields['playback.crf']).toContain('menor');
		expect(collected.fields['ui.server_name']).toContain('al menos');
		expect(collected.values['playback.crf']).toBe('99');
		expect(collected.values['ui.server_name']).toBe('');
		expect(collected.changes).toEqual({ 'scan.interval': '30m' });
	});

	it('encuentra una sección del catálogo', () => {
		const response = {
			sections: [{ key: 'scan', label: 'Escaneo', settings: [interval] }]
		};
		expect(findSettingsSection(response, 'scan')?.label).toBe('Escaneo');
		expect(findSettingsSection(response, 'ui')).toBeNull();
	});
});
