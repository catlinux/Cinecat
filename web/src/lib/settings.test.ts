import { describe, expect, it } from 'vitest';
import type { Setting } from './api';
import {
	collectSettingsForm,
	findSettingsSection,
	formatDurationText,
	formatSettingDefault,
	isNumericEnum,
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

const pageSize = setting({
	key: 'ui.page_size',
	type: 'enum',
	default: 48,
	value: 48,
	options: [
		{ value: '24', label: '24' },
		{ value: '48', label: '48' },
		{ value: '96', label: '96' }
	],
	min: 24,
	max: 96
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

describe('formato de ajustes', () => {
	it('distingue los enum numéricos', () => {
		expect(isNumericEnum(pageSize)).toBe(true);
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
		expect(formatSettingDefault(pageSize)).toBe('48');
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
		expect(parseSettingValue(pageSize, '24')).toEqual({ value: 24 });
		expect(parseSettingValue(language, 'en-US')).toEqual({ value: 'en-US' });
	});

	it('devuelve errores de tipo, rango y opción', () => {
		expect(parseSettingValue(crf, '2.5').error).toContain('entero');
		expect(parseSettingValue(crf, '17').error).toContain('mayor');
		expect(parseSettingValue(serverName, '').error).toContain('al menos');
		expect(parseSettingValue(interval, 'mañana').error).toContain('Duración');
		expect(parseSettingValue(pageSize, '50').error).toContain('opción');
		expect(parseSettingValue(language, 'fr-FR').error).toContain('opción');
	});
});

describe('recogida del formulario', () => {
	it('envía solo los campos modificados y conserva el tipo', () => {
		const form = new FormData();
		form.set('ui.page_size', '24');
		form.set('metadata.language', 'es-ES');
		form.set('ui.server_name', 'Cinecat');
		form.set('scan.interval', '30m');
		form.set('playback.crf', '20');
		form.set('metadata.match_threshold', '0.75');

		const collected = collectSettingsForm(
			[pageSize, language, serverName, interval, crf, threshold, showSeries],
			form
		);

		expect(collected.changes).toEqual({
			'ui.page_size': 24,
			'scan.interval': '30m',
			'ui.show_series': false
		});
		expect(collected.fields).toEqual({});
		expect(collected.values['ui.page_size']).toBe('24');
		expect(collected.values['ui.show_series']).toBe(false);
	});

	it('no envía nada si los valores no cambian', () => {
		const form = new FormData();
		form.set('ui.page_size', '48');
		form.set('metadata.language', 'es-ES');
		form.set('ui.server_name', 'Cinecat');
		form.set('scan.interval', '6h');
		form.set('playback.crf', '20');
		form.set('metadata.match_threshold', '0.75');
		form.set('ui.show_series', '1');

		const collected = collectSettingsForm(
			[pageSize, language, serverName, interval, crf, threshold, showSeries],
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
