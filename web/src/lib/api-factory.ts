import { ApiClient } from './api';
import { apiBaseUrl } from './config';

// Error de configuración del servidor web (no de la API).
export class ConfigurationError extends Error {
	constructor(message: string) {
		super(message);
		this.name = 'ConfigurationError';
	}
}

// Crea el cliente de API del lado servidor con el fetch del evento de SvelteKit.
export function createApiClient(fetchImpl: typeof fetch): ApiClient {
	if (!apiBaseUrl) {
		throw new ConfigurationError('Falta configurar PUBLIC_API_URL en el servidor.');
	}
	return new ApiClient({ baseUrl: apiBaseUrl, fetch: fetchImpl });
}
