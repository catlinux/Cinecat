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

export interface PlaybackProgress {
	position_seconds: number;
	duration_seconds: number;
	updated_at: string;
}

export interface ItemCollection {
	id: number;
	name: string;
}

export interface CastCredit {
	name: string;
	character: string;
	photo_url: string;
}

export interface CrewCredit {
	name: string;
	job: string;
	photo_url: string;
}

export interface Credits {
	cast: CastCredit[];
	crew: CrewCredit[];
}

export interface ItemPreferences {
	audio_index: number | null;
	subtitle_id: string | null;
	quality: string | null;
}

export interface MediaItem {
	id: number;
	library_id: number;
	path: string;
	title: string;
	file_title: string;
	year: number | null;
	kind: 'movie' | 'series' | 'episode';
	season: number | null;
	episode: number | null;
	tmdb_id: number | null;
	overview: string | null;
	poster_url: string | null;
	poster_custom: boolean;
	original_title: string | null;
	imdb_id: string | null;
	tagline: string | null;
	genres: string[];
	runtime_minutes: number | null;
	rating: number | null;
	directors: string[];
	cast: string[];
	credits?: Credits | null;
	backdrop_url: string | null;
	backdrop_custom: boolean;
	overview_language: string | null;
	match_status: 'auto' | 'manual' | 'none' | null;
	match_score: number | null;
	watched: boolean;
	progress: PlaybackProgress | null;
	preferences?: ItemPreferences | null;
	collections: ItemCollection[];
	series_id: number | null;
	series_title: string | null;
	episode_title: string | null;
	still_url: string | null;
}

export type CollectionSource = 'tmdb' | 'manual';

export interface Collection {
	id: number;
	name: string;
	overview: string | null;
	poster_url: string | null;
	backdrop_url: string | null;
	item_count: number;
	source: CollectionSource;
	edited: boolean;
}

export interface CollectionsResponse {
	collections: Collection[];
	total: number;
	limit: number;
	offset: number;
}

export interface CollectionDetail extends Collection {
	items: MediaItem[];
}

export type CollectionSort = 'name' | 'added' | 'size';

export interface CollectionsQuery {
	limit?: number;
	offset?: number;
	q?: string;
	sort?: CollectionSort;
}

export interface CollectionUpdate {
	name?: string;
	overview?: string;
	poster_url?: string;
	backdrop_url?: string;
}

export interface HomeCounts {
	movies: number;
	episodes: number;
	series: number;
}

export interface HomeResponse {
	continue: MediaItem[];
	next_up: MediaItem[];
	recent_movies: MediaItem[];
	counts: HomeCounts;
}

export interface Candidate {
	tmdb_id: number;
	title: string;
	original_title: string;
	year: number;
	overview: string;
	poster_url: string;
	popularity: number;
	vote_count: number;
}

export interface CandidatesResponse {
	candidates: Candidate[];
}

export interface PlaybackInfo {
	mode: 'direct' | 'hls' | '';
	description: string;
	supported: boolean;
}

export interface VideoStream {
	index: number;
	codec: string;
	codec_long_name: string;
	profile: string;
	width: number;
	height: number;
	display_aspect_ratio: string;
	frame_rate: number;
	bit_rate: number;
	pixel_format: string;
	bit_depth: number;
	hdr: string;
	language: string;
	title: string;
	default: boolean;
}

export interface AudioStream {
	index: number;
	codec: string;
	codec_long_name: string;
	profile: string;
	channels: number;
	channel_layout: string;
	sample_rate: number;
	bit_rate: number;
	language: string;
	title: string;
	default: boolean;
}

export interface SubtitleStream {
	index: number;
	codec: string;
	language: string;
	title: string;
	forced: boolean;
	default: boolean;
	text: boolean;
}

export interface FileInfo {
	container: string;
	container_long_name: string;
	size_bytes: number;
	duration_seconds: number;
	bit_rate: number;
	modified_at: string;
	file_name: string;
	path: string;
	video: VideoStream[];
	audio: AudioStream[];
	subtitles: SubtitleStream[];
	chapters: number;
	quality: string;
	playback: PlaybackInfo;
}

export type FileInfoError = 'not_found' | 'unavailable' | 'failed';

export type FileInfoResult =
	| { data: FileInfo; error?: null }
	| { data?: null; error: FileInfoError };

export interface ImageOption {
	url: string;
	full_url: string;
	language: string;
	vote_average: number;
	width: number;
	height: number;
}

export interface ImageOptions {
	posters: ImageOption[];
	backdrops: ImageOption[];
}

export type ImageKind = 'poster' | 'backdrop';

export interface GenreCount {
	name: string;
	count: number;
}

export interface GenresResponse {
	genres: GenreCount[];
}

export interface ItemsResponse {
	items: MediaItem[];
	total: number;
	limit: number;
	offset: number;
}

export interface SimilarItemsResponse {
	items: MediaItem[];
}

export interface SimilarSeriesResponse {
	series: Series[];
}

export interface ItemsQuery {
	limit?: number;
	offset?: number;
	q?: string;
	kind?: 'movie' | 'episode';
	unmatched?: boolean;
}

export type SeriesSort = 'title' | 'year' | 'added';

export interface Series {
	id: number;
	title: string;
	file_title: string;
	year: number | null;
	poster_url: string | null;
	genres: string[];
	season_count: number;
	episode_count: number;
	watched_count: number;
	match_status: 'auto' | 'manual' | 'none' | null;
}

export interface SeriesResponse {
	series: Series[];
	total: number;
	limit: number;
	offset: number;
}

export interface SeriesQuery {
	limit?: number;
	offset?: number;
	q?: string;
	sort?: SeriesSort;
	letter?: string;
	genre?: string;
	unmatched?: boolean;
}

export interface SeriesSeason {
	season: number;
	name: string;
	overview: string | null;
	poster_url: string | null;
	episode_count: number;
	watched_count: number;
}

export interface SeriesDetail {
	id: number;
	title: string;
	file_title: string;
	folder: string;
	original_title: string | null;
	year: number | null;
	tagline: string | null;
	overview: string | null;
	overview_language: string | null;
	poster_url: string | null;
	backdrop_url: string | null;
	genres: string[];
	rating: number | null;
	directors: string[];
	cast: string[];
	credits?: Credits | null;
	tmdb_id: number | null;
	match_status: 'auto' | 'manual' | 'none' | null;
	match_score: number | null;
	poster_custom: boolean;
	backdrop_custom: boolean;
	season_count: number;
	episode_count: number;
	watched_count: number;
	seasons: SeriesSeason[];
	next_episode: MediaItem | null;
}

export interface SeriesSeasonDetail {
	season: number;
	name: string;
	overview: string | null;
	poster_url: string | null;
	items: MediaItem[];
}

export interface PlaybackAudioTrack {
	index: number;
	language: string;
	language_name: string;
	codec: string;
	channels: number;
	channel_layout: string;
	title: string;
	default: boolean;
}

export interface PlaybackQuality {
	value: string;
	label: string;
}

export interface PlaybackSubtitle {
	id: string;
	source: 'embedded' | 'external';
	language: string;
	language_name: string;
	title: string;
	codec: string;
	forced: boolean;
	default: boolean;
	url: string;
}

export interface PlayResponse {
	url: string;
	mode: 'direct' | 'remux';
	expires_at: string;
	audio_tracks?: PlaybackAudioTrack[];
	selected_audio?: number;
	qualities?: PlaybackQuality[];
	selected_quality?: string;
	subtitles?: PlaybackSubtitle[];
	selected_subtitle?: string | null;
}

export interface PlayQuery {
	audio?: number;
	quality?: string;
	subtitle?: string;
}

export interface ScanStatus {
	running: boolean;
	started_at: string | null;
	finished_at: string | null;
	libraries: number;
	files: number;
	upserted: number;
	identified: number;
	unidentified: number;
	series_identified?: number;
	series_unidentified?: number;
	warnings: string[];
	error: string;
}

export type SettingType = 'bool' | 'int' | 'float' | 'string' | 'duration' | 'enum';

export interface SettingOption {
	value: string;
	label: string;
}

export interface Setting {
	key: string;
	label: string;
	description: string;
	type: SettingType;
	options: SettingOption[];
	min: number;
	max: number;
	default: unknown;
	value: unknown;
	modified: boolean;
}

export interface SettingsSection {
	key: string;
	label: string;
	settings: Setting[];
}

export interface SettingsResponse {
	sections: SettingsSection[];
}

export type SettingsValues = Record<string, string | boolean>;
export type SettingsChanges = Record<string, unknown>;

export interface ScanStarted {
	started: boolean;
}

export interface ServerVersion {
	server: string;
	commit: string;
	built_at: string;
}

export interface CurrentUserResponse {
	username: string;
	role: 'admin' | 'user';
}

// Error del servidor o de red. status 0 significa que no se pudo contactar.
export class ApiError extends Error {
	readonly status: number;
	readonly fields: Record<string, string> | null;

	constructor(message: string, status: number, fields: Record<string, string> | null = null) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.fields = fields;
	}
}

export interface ApiClientOptions {
	baseUrl: string;
	fetch?: typeof fetch;
}

interface RequestOptions {
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
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

	async startScan(token: string): Promise<ScanStarted> {
		return this.post<ScanStarted>('/api/v1/libraries/scan', undefined, token);
	}

	async getScanStatus(token: string): Promise<ScanStatus> {
		return this.get<ScanStatus>('/api/v1/libraries/scan', token);
	}

	async getServerVersion(): Promise<ServerVersion> {
		return this.get<ServerVersion>('/api/v1/system/version');
	}

	async getSettings(token: string): Promise<SettingsResponse> {
		return this.get<SettingsResponse>('/api/v1/settings', token);
	}

	async patchSettings(changes: SettingsChanges, token: string): Promise<SettingsResponse> {
		return this.request<SettingsResponse>('/api/v1/settings', {
			method: 'PATCH',
			body: changes,
			token
		});
	}

	async resetSetting(key: string, token: string): Promise<Setting> {
		return this.request<Setting>(`/api/v1/settings/${encodeURIComponent(key)}`, {
			method: 'DELETE',
			token
		});
	}

	async getItem(id: number, token: string): Promise<MediaItem> {
		return this.get<MediaItem>(`/api/v1/items/${id}`, token);
	}

	async getItems(query: ItemsQuery, token: string): Promise<ItemsResponse> {
		const params = new URLSearchParams();
		if (query.limit !== undefined) {
			params.set('limit', String(query.limit));
		}
		if (query.offset !== undefined) {
			params.set('offset', String(query.offset));
		}
		if (query.q) {
			params.set('q', query.q);
		}
		if (query.kind) {
			params.set('kind', query.kind);
		}
		if (query.unmatched) {
			params.set('unmatched', 'true');
		}
		const queryString = params.toString();
		const suffix = queryString ? `?${queryString}` : '';
		return this.get<ItemsResponse>(`/api/v1/items${suffix}`, token);
	}

	async getCandidates(
		id: number,
		query: string,
		year: number,
		token: string
	): Promise<CandidatesResponse> {
		const params = new URLSearchParams({ q: query });
		if (year > 0) {
			params.set('year', String(year));
		}
		return this.get<CandidatesResponse>(`/api/v1/items/${id}/candidates?${params}`, token);
	}

	async setWatched(id: number, watched: boolean, token: string): Promise<void> {
		const path = `/api/v1/items/${id}/watched`;
		if (watched) {
			await this.put<void>(path, undefined, token);
			return;
		}
		await this.delete(path, token);
	}

	async matchItem(id: number, tmdbID: number | null, token: string): Promise<MediaItem> {
		return this.put<MediaItem>(
			`/api/v1/items/${id}/match`,
			{ tmdb_id: tmdbID },
			token
		);
	}

	async resetMatch(id: number, token: string): Promise<MediaItem> {
		return this.request<MediaItem>(`/api/v1/items/${id}/match`, {
			method: 'DELETE',
			token
		});
	}

	async getHome(token: string): Promise<HomeResponse> {
		return this.get<HomeResponse>('/api/v1/home', token);
	}

	async getSeries(query: SeriesQuery, token: string): Promise<SeriesResponse> {
		const params = new URLSearchParams();
		if (query.limit !== undefined) {
			params.set('limit', String(query.limit));
		}
		if (query.offset !== undefined) {
			params.set('offset', String(query.offset));
		}
		if (query.q) {
			params.set('q', query.q);
		}
		if (query.sort) {
			params.set('sort', query.sort);
		}
		if (query.letter) {
			params.set('letter', query.letter);
		}
		if (query.genre) {
			params.set('genre', query.genre);
		}
		if (query.unmatched) {
			params.set('unmatched', 'true');
		}
		const queryString = params.toString();
		const suffix = queryString ? `?${queryString}` : '';
		return this.get<SeriesResponse>(`/api/v1/series${suffix}`, token);
	}

	async getSeriesGenres(token: string): Promise<GenresResponse> {
		return this.get<GenresResponse>('/api/v1/series/genres', token);
	}

	async getSeriesDetail(id: number, token: string): Promise<SeriesDetail> {
		return this.get<SeriesDetail>(`/api/v1/series/${id}`, token);
	}

	async getSeriesSeason(id: number, season: number, token: string): Promise<SeriesSeasonDetail> {
		return this.get<SeriesSeasonDetail>(`/api/v1/series/${id}/seasons/${season}`, token);
	}

	async getSeriesCandidates(
		id: number,
		query: string,
		year: number,
		token: string
	): Promise<CandidatesResponse> {
		const params = new URLSearchParams({ q: query });
		if (year > 0) {
			params.set('year', String(year));
		}
		return this.get<CandidatesResponse>(`/api/v1/series/${id}/candidates?${params}`, token);
	}

	async matchSeries(
		id: number,
		tmdbID: number | null,
		token: string
	): Promise<SeriesDetail> {
		return this.put<SeriesDetail>(
			`/api/v1/series/${id}/match`,
			{ tmdb_id: tmdbID },
			token
		);
	}

	async resetSeriesMatch(id: number, token: string): Promise<SeriesDetail> {
		return this.request<SeriesDetail>(`/api/v1/series/${id}/match`, {
			method: 'DELETE',
			token
		});
	}

	async getSeriesImageOptions(id: number, token: string): Promise<ImageOptions> {
		return this.get<ImageOptions>(`/api/v1/series/${id}/images`, token);
	}

	async setSeriesImage(
		id: number,
		kind: ImageKind,
		url: string,
		token: string
	): Promise<SeriesDetail> {
		return this.put<SeriesDetail>(`/api/v1/series/${id}/images/${kind}`, { url }, token);
	}

	async resetSeriesImage(id: number, kind: ImageKind, token: string): Promise<SeriesDetail> {
		return this.request<SeriesDetail>(`/api/v1/series/${id}/images/${kind}`, {
			method: 'DELETE',
			token
		});
	}

	async setSeriesWatched(id: number, watched: boolean, token: string): Promise<void> {
		const path = `/api/v1/series/${id}/watched`;
		if (watched) {
			await this.put<void>(path, undefined, token);
			return;
		}
		await this.delete(path, token);
	}

	async setSeriesSeasonWatched(
		id: number,
		season: number,
		watched: boolean,
		token: string
	): Promise<void> {
		const path = `/api/v1/series/${id}/seasons/${season}/watched`;
		if (watched) {
			await this.put<void>(path, undefined, token);
			return;
		}
		await this.delete(path, token);
	}

	async putProgress(
		id: number,
		positionSeconds: number,
		durationSeconds: number,
		token: string
	): Promise<void> {
		await this.put<void>(
			`/api/v1/items/${id}/progress`,
			{
				position_seconds: positionSeconds,
				duration_seconds: durationSeconds
			},
			token
		);
	}

	async deleteProgress(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/items/${id}/progress`, token);
	}

	async getSimilarItems(id: number, limit: number, token: string): Promise<SimilarItemsResponse> {
		return this.get<SimilarItemsResponse>(`/api/v1/items/${id}/similar?limit=${limit}`, token);
	}

	async getSimilarSeries(id: number, limit: number, token: string): Promise<SimilarSeriesResponse> {
		return this.get<SimilarSeriesResponse>(`/api/v1/series/${id}/similar?limit=${limit}`, token);
	}

	async getPlay(id: number, query: PlayQuery, token: string): Promise<PlayResponse> {
		const params = new URLSearchParams();
		if (query.audio !== undefined) {
			params.set('audio', String(query.audio));
		}
		if (query.quality) {
			params.set('quality', query.quality);
		}
		if (query.subtitle !== undefined) {
			params.set('subtitle', query.subtitle);
		}
		const suffix = params.toString() ? `?${params}` : '';
		return this.get<PlayResponse>(`/api/v1/items/${id}/play${suffix}`, token);
	}

	async getPreferences(id: number, token: string): Promise<ItemPreferences> {
		return this.get<ItemPreferences>(`/api/v1/items/${id}/preferences`, token);
	}

	async putPreferences(id: number, changes: Partial<ItemPreferences>, token: string): Promise<void> {
		await this.put<void>(`/api/v1/items/${id}/preferences`, changes, token);
	}

	async deletePreferences(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/items/${id}/preferences`, token);
	}

	async getFileInfo(id: number, token: string): Promise<FileInfo> {
		return this.get<FileInfo>(`/api/v1/items/${id}/file-info`, token);
	}

	async getImageOptions(id: number, token: string): Promise<ImageOptions> {
		return this.get<ImageOptions>(`/api/v1/items/${id}/images`, token);
	}

	async setImage(
		id: number,
		kind: ImageKind,
		url: string,
		token: string
	): Promise<MediaItem> {
		return this.put<MediaItem>(`/api/v1/items/${id}/images/${kind}`, { url }, token);
	}

	async resetImage(id: number, kind: ImageKind, token: string): Promise<MediaItem> {
		return this.request<MediaItem>(`/api/v1/items/${id}/images/${kind}`, {
			method: 'DELETE',
			token
		});
	}

	async getCollections(query: CollectionsQuery, token: string): Promise<CollectionsResponse> {
		const params = new URLSearchParams();
		if (query.limit !== undefined) {
			params.set('limit', String(query.limit));
		}
		if (query.offset !== undefined) {
			params.set('offset', String(query.offset));
		}
		if (query.q) {
			params.set('q', query.q);
		}
		if (query.sort) {
			params.set('sort', query.sort);
		}
		const queryString = params.toString();
		const suffix = queryString ? `?${queryString}` : '';
		return this.get<CollectionsResponse>(`/api/v1/collections${suffix}`, token);
	}

	async getCollection(id: number, token: string): Promise<CollectionDetail> {
		return this.get<CollectionDetail>(`/api/v1/collections/${id}`, token);
	}

	async createCollection(name: string, overview: string, token: string): Promise<Collection> {
		return this.post<Collection>('/api/v1/collections', { name, overview }, token);
	}

	async updateCollection(
		id: number,
		changes: CollectionUpdate,
		token: string
	): Promise<Collection> {
		return this.request<Collection>(`/api/v1/collections/${id}`, {
			method: 'PATCH',
			body: changes,
			token
		});
	}

	async deleteCollection(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/collections/${id}`, token);
	}

	async addCollectionItem(id: number, mediaItemID: number, token: string): Promise<void> {
		await this.post<void>(
			`/api/v1/collections/${id}/items`,
			{ media_item_id: mediaItemID },
			token
		);
	}

	async removeCollectionItem(id: number, mediaItemID: number, token: string): Promise<void> {
		await this.delete(`/api/v1/collections/${id}/items/${mediaItemID}`, token);
	}

	async reorderCollection(id: number, mediaItemIDs: number[], token: string): Promise<void> {
		await this.put<void>(
			`/api/v1/collections/${id}/order`,
			{ media_item_ids: mediaItemIDs },
			token
		);
	}

	async get<T>(path: string, token?: string): Promise<T> {
		return this.request<T>(path, { token });
	}

	async post<T>(path: string, body?: unknown, token?: string): Promise<T> {
		return this.request<T>(path, { method: 'POST', body, token });
	}

	async put<T>(path: string, body?: unknown, token?: string): Promise<T> {
		return this.request<T>(path, { method: 'PUT', body, token });
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
		let fields: Record<string, string> | null = null;
		try {
			const data = (await response.json()) as Partial<ErrorResponse> & {
				fields?: unknown;
			};
			if (typeof data.error === 'string' && data.error.trim() !== '') {
				serverMessage = data.error.trim();
			}
			if (data.fields && typeof data.fields === 'object' && !Array.isArray(data.fields)) {
				fields = {};
				for (const [key, value] of Object.entries(data.fields)) {
					if (typeof value === 'string') {
						fields[key] = value;
					}
				}
				if (Object.keys(fields).length === 0) {
					fields = null;
				}
			}
		} catch {
			// Respuesta sin JSON: se usa el mensaje por defecto.
		}
		return new ApiError(serverMessage || defaultMessage(response.status), response.status, fields);
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
