export type ThemePreference = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export const THEME_STORAGE_KEY = 'cinecat-theme';

// La preferencia vive solo en el navegador (localStorage): no es un dato de
// sesión ni de servidor.
export function parseThemePreference(value: string | null | undefined): ThemePreference {
	return value === 'light' || value === 'dark' || value === 'system' ? value : 'system';
}

export function resolveTheme(
	preference: ThemePreference,
	prefersDark: boolean
): ResolvedTheme {
	if (preference === 'system') {
		return prefersDark ? 'dark' : 'light';
	}
	return preference;
}

export function systemPrefersDark(): boolean {
	return (
		typeof window !== 'undefined' &&
		typeof window.matchMedia === 'function' &&
		window.matchMedia('(prefers-color-scheme: dark)').matches
	);
}

export function getStoredTheme(): ThemePreference {
	if (typeof localStorage === 'undefined') {
		return 'system';
	}
	try {
		return parseThemePreference(localStorage.getItem(THEME_STORAGE_KEY));
	} catch {
		return 'system';
	}
}

export function applyTheme(preference: ThemePreference): void {
	if (typeof document === 'undefined') {
		return;
	}
	const resolved = resolveTheme(preference, systemPrefersDark());
	const root = document.documentElement;
	root.dataset.theme = resolved;
	root.style.colorScheme = resolved;
}

export function setThemePreference(preference: ThemePreference): void {
	try {
		localStorage.setItem(THEME_STORAGE_KEY, preference);
	} catch {
		// Sin almacenamiento (modo privado estricto): el tema vale para esta página.
	}
	applyTheme(preference);
}
