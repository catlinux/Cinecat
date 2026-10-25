import { ApiClient } from './api';
import { apiBaseUrl } from './config';

// Error de configuración del servidor web (no de la API). Se mantiene el
// tipo por compatibilidad con quien lo importe, aunque hoy no se lance:
// PUBLIC_API_URL vacío es válido (rutas relativas contra el mismo proxy).
export class ConfigurationError extends Error {
	constructor(message: string) {
		super(message);
		this.name = 'ConfigurationError';
	}
}

// Crea el cliente de API del lado servidor con el fetch del evento de SvelteKit.
export function createApiClient(fetchImpl: typeof fetch): ApiClient {
	return new ApiClient({ baseUrl: apiBaseUrl, fetch: fetchImpl });
}
