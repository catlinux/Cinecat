import { describe, expect, it } from 'vitest';
import { parseThemePreference, resolveTheme } from './theme';

describe('theme', () => {
	it('acepta solo las preferencias válidas y cae en system', () => {
		expect(parseThemePreference('light')).toBe('light');
		expect(parseThemePreference('dark')).toBe('dark');
		expect(parseThemePreference('system')).toBe('system');
		expect(parseThemePreference('otro')).toBe('system');
		expect(parseThemePreference(null)).toBe('system');
	});

	it('resuelve system según prefers-color-scheme', () => {
		expect(resolveTheme('system', true)).toBe('dark');
		expect(resolveTheme('system', false)).toBe('light');
		expect(resolveTheme('dark', false)).toBe('dark');
		expect(resolveTheme('light', true)).toBe('light');
	});
});
