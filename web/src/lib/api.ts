export interface LoginRequest {
	username: string;
	password: string;
}

export interface LoginResponse {
	token: string;
	expires_at: string;
}

export interface ErrorResponse {
	error: string;
}

export interface Library {
	id: number;
	name: string;
	path: string;
}

export interface LibrariesResponse {
	libraries: Library[];
}

export interface MediaItem {
	id: number;
	library_id: number;
	path: string;
	title: string;
	year: number | null;
	kind: 'movie' | 'series' | 'episode';
	season: number | null;
	episode: number | null;
	tmdb_id: number | null;
	overview: string | null;
	poster_url: string | null;
}

export interface ItemsResponse {
	items: MediaItem[];
	total: number;
	limit: number;
	offset: number;
}

export interface PlayResponse {
	url: string;
	mode: 'direct' | 'remux';
	expires_at: string;
}

export interface ScanResult {
	libraries: number;
	files: number;
	upserted: number;
	warnings: string[];
}

export interface CurrentUserResponse {
	username: string;
	role: 'admin' | 'user';
}

// Error del servidor o de red. status 0 significa que no se pudo contactar.
export class ApiError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

export interface ApiClientOptions {
	baseUrl: string;
	fetch?: typeof fetch;
}

interface RequestOptions {
	method?: 'GET' | 'POST' | 'DELETE';
	body?: unknown;
	token?: string;
	empty?: boolean;
}

// Cliente tipado contra api/openapi.yaml. Los tipos se escriben a mano para no
// añadir un generador de código solo por este paso.
export class ApiClient {
	private readonly baseUrl: string;
	private readonly fetchImpl: typeof fetch;

	constructor(options: ApiClientOptions) {
		this.baseUrl = options.baseUrl.replace(/\/+$/, '');
		this.fetchImpl = options.fetch ?? globalThis.fetch.bind(globalThis);
	}

	async login(credentials: LoginRequest): Promise<LoginResponse> {
		return this.request<LoginResponse>('/api/v1/auth/login', {
			method: 'POST',
			body: credentials
		});
	}

	async logout(token: string): Promise<void> {
		await this.request<void>('/api/v1/auth/logout', { method: 'POST', token, empty: true });
	}

	async changePassword(
		currentPassword: string,
		newPassword: string,
		token: string
	): Promise<void> {
		await this.request<void>('/api/v1/auth/password', {
			method: 'POST',
			body: { current_password: currentPassword, new_password: newPassword },
			token,
			empty: true
		});
	}

	async getCurrentUser(token: string): Promise<CurrentUserResponse> {
		return this.get<CurrentUserResponse>('/api/v1/auth/me', token);
	}

	async listLibraries(token: string): Promise<LibrariesResponse> {
		return this.get<LibrariesResponse>('/api/v1/libraries', token);
	}

	async createLibrary(name: string, path: string, token: string): Promise<Library> {
		return this.post<Library>('/api/v1/libraries', { name, path }, token);
	}

	async deleteLibrary(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/libraries/${id}`, token);
	}

	async scanLibraries(token: string): Promise<ScanResult> {
		return this.post<ScanResult>('/api/v1/libraries/scan', undefined, token);
	}

	async get<T>(path: string, token?: string): Promise<T> {
		return this.request<T>(path, { token });
	}

	async post<T>(path: string, body?: unknown, token?: string): Promise<T> {
		return this.request<T>(path, { method: 'POST', body, token });
	}

	async delete(path: string, token?: string): Promise<void> {
		await this.request<void>(path, { method: 'DELETE', token, empty: true });
	}

	private async request<T>(path: string, options: RequestOptions): Promise<T> {
		const headers: Record<string, string> = { Accept: 'application/json' };
		if (options.body !== undefined) {
			headers['Content-Type'] = 'application/json';
		}
		if (options.token) {
			headers.Authorization = `Bearer ${options.token}`;
		}

		let response: Response;
		try {
			response = await this.fetchImpl(this.url(path), {
				method: options.method ?? 'GET',
				headers,
				body: options.body === undefined ? undefined : JSON.stringify(options.body)
			});
		} catch {
			throw new ApiError('No se pudo conectar con el servidor de Cinecat.', 0);
		}

		if (!response.ok) {
			throw await this.errorFrom(response);
		}
		if (options.empty || response.status === 204) {
			return undefined as T;
		}

		try {
			return (await response.json()) as T;
		} catch {
			throw new ApiError('El servidor devolvió una respuesta no válida.', response.status);
		}
	}

	private url(path: string): string {
		return `${this.baseUrl}${path.startsWith('/') ? path : `/${path}`}`;
	}

	private async errorFrom(response: Response): Promise<ApiError> {
		let serverMessage = '';
		try {
			const data = (await response.json()) as Partial<ErrorResponse>;
			if (typeof data.error === 'string' && data.error.trim() !== '') {
				serverMessage = data.error.trim();
			}
		} catch {
			// Respuesta sin JSON: se usa el mensaje por defecto.
		}
		return new ApiError(serverMessage || defaultMessage(response.status), response.status);
	}
}

function defaultMessage(status: number): string {
	switch (status) {
		case 400:
			return 'Petición no válida.';
		case 401:
			return 'No autorizado.';
		case 403:
			return 'No tienes permiso para esta acción.';
		case 404:
			return 'No encontrado.';
		case 413:
			return 'La petición es demasiado grande.';
		case 429:
			return 'Demasiadas peticiones. Inténtalo más tarde.';
		default:
			return `Error del servidor (${status}).`;
	}
}
