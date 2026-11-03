import { describe, expect, it } from 'vitest';
import type { ItemSegment } from './api';
import {
	emptySegmentDrafts,
	formatSegmentTime,
	parseSegmentForm,
	parseSegmentTime,
	segmentDrafts
} from './segments';

describe('tiempos de segmentos', () => {
	it('formatea mm:ss y h:mm:ss', () => {
		expect(formatSegmentTime(0)).toBe('00:00');
		expect(formatSegmentTime(65)).toBe('01:05');
		expect(formatSegmentTime(3723)).toBe('1:02:03');
		expect(formatSegmentTime(-1)).toBe('');
		expect(formatSegmentTime(Number.NaN)).toBe('');
	});

	it('parsea mm:ss y h:mm:ss', () => {
		expect(parseSegmentTime('00:00')).toBe(0);
		expect(parseSegmentTime('01:05')).toBe(65);
		expect(parseSegmentTime('1:02:03')).toBe(3723);
		expect(parseSegmentTime(' 1:02:03 ')).toBe(3723);
		expect(parseSegmentTime('1:60')).toBeNull();
		expect(parseSegmentTime('1:2:03')).toBeNull();
		expect(parseSegmentTime('')).toBeNull();
		expect(parseSegmentTime('abc')).toBeNull();
	});

	it('rellena las filas con los segmentos manuales', () => {
		const manual: ItemSegment[] = [
			{ kind: 'intro', start: 12, end: 42 },
			{ kind: 'credits', start: 3600, end: 3665 }
		];
		expect(segmentDrafts(manual)).toEqual({
			intro: { start: '00:12', end: '00:42' },
			credits: { start: '1:00:00', end: '1:01:05' }
		});
		expect(segmentDrafts(null)).toEqual(emptySegmentDrafts());
	});

	it('convierte las filas y permite vaciar una fila', () => {
		const values = {
			intro: { start: '00:10', end: '00:25' },
			credits: { start: '', end: '' }
		};
		expect(parseSegmentForm(values)).toEqual({
			segments: [{ kind: 'intro', start: 10, end: 25 }],
			values,
			error: null
		});
		expect(
			parseSegmentForm({
				intro: { start: '', end: '00:25' },
				credits: { start: '00:30', end: '00:40' }
			}).segments
		).toEqual([{ kind: 'credits', start: 30, end: 40 }]);
	});

	it('rechaza tiempos inválidos y un fin anterior al inicio', () => {
		const invalid = parseSegmentForm({
			intro: { start: '00:10', end: '00:xx' },
			credits: { start: '', end: '' }
		});
		expect(invalid.error).toContain('Intro');
		expect(invalid.segments).toEqual([]);
		expect(
			parseSegmentForm({
				intro: { start: '00:30', end: '00:10' },
				credits: { start: '', end: '' }
			}).error
		).toContain('posterior');
	});
});
