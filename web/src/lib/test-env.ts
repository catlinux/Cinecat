// Sustituto de $env/dynamic/public solo para las pruebas con Vitest.
export const env: Record<string, string | undefined> = {
	PUBLIC_API_URL: 'http://api.test'
};
