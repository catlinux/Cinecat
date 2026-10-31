export interface LoginRequest {
	username: string;
	password: string;
}

export interface LoginResponse {
	token: string;
	expires_at: string;
	session_id: number;
}

export type SessionClient = 'web' | 'android' | 'androidtv';

export interface Session {
	id: number;
	device_name: string;
	client: SessionClient;
	created_at: string;
	last_seen_at: string;
	expires_at: string;
	current: boolean;
}

export interface SessionsResponse {
	sessions: Session[];
}

export interface PairPreview {
	device_name: string;
	client: 'android' | 'androidtv';
	expires_at?: string;
}

export interface ErrorResponse {
	error: string;
}

export type LibraryKind = 'video' | 'clips' | 'music';

export interface LibraryPath {
	id: number;
	path: string;
}

export interface Library {
	id: number;
	name: string;
	kind: LibraryKind;
	path?: string;
	paths?: LibraryPath[];
	item_count?: number;
}

export interface LibrariesResponse {
	libraries: Library[];
}

export interface UploadTargetRoot {
	library_id: number;
	library_name: string;
	library_kind: 'video' | 'clips';
	path: string;
	movie_count: number;
	series_count: number;
}

export interface UploadTargetsResponse {
	allowed: boolean;
	roots: UploadTargetRoot[];
}

export interface SeriesUploadResponse {
	path: string;
	size: number;
	season: number;
	episode: number;
}

export interface CreateLibraryInput {
	name: string;
	kind: LibraryKind;
	paths: string[];
}

export interface FSRoot {
	name: string;
	path: string;
}

export interface FSEntry {
	name: string;
	path: string;
}

export interface FSRootsResponse {
	roots: FSRoot[];
}

export interface FSBrowseResponse {
	path: string;
	parent: string | null;
	entries: FSEntry[];
	truncated: boolean;
}

export type FSResponse = FSRootsResponse | FSBrowseResponse;

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
	id?: number;
	name: string;
	character: string;
	photo_url: string;
}

export interface CrewCredit {
	id?: number;
	name: string;
	job: string;
	photo_url: string;
}

export interface Person {
	id: number;
	name: string;
	biography: string;
	birthday: string;
	deathday: string;
	place_of_birth: string;
	photo_url: string;
	known_for: string;
}

export interface PersonMedia {
	type: 'movie' | 'series';
	id?: number;
	tmdb_id?: number;
	title: string;
	year: number | null;
	poster_url: string | null;
	rating: number | null;
	roles: string[];
}

export interface PersonResponse {
	person: Person;
	in_library: PersonMedia[];
	not_in_library: PersonMedia[];
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
	library_kind?: LibraryKind;
	path?: string;
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
	trailer_url?: string | null;
	trailer_language?: 'es' | 'en' | null;
	directors: string[];
	cast?: string[];
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
	favorite?: boolean;
	watchlist?: boolean;
	series_id: number | null;
	series_title: string | null;
	episode_title: string | null;
	still_url: string | null;
	series_poster_url?: string | null;
	series_backdrop_url?: string | null;
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
	favorite?: boolean;
	watchlist?: boolean;
}

export interface CollectionsResponse {
	collections: Collection[];
	total: number;
	limit: number;
	offset: number;
}

export interface MissingMovie {
	tmdb_id: number;
	title: string;
	year: number;
	poster_url: string;
}

export interface CollectionDetail extends Collection {
	tmdb_collection_id: number | null;
	missing: MissingMovie[];
	items: MediaItem[];
}

export interface CollectionCandidate {
	tmdb_id: number;
	name: string;
	overview: string;
	poster_url: string;
}

export interface CollectionCandidatesResponse {
	candidates: CollectionCandidate[];
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
	recently_watched?: MediaItem[];
	recent_series?: Series[];
	has_clips?: boolean;
	has_music?: boolean;
	watchlist?: HomeListEntry[];
	favorites?: HomeListEntry[];
	counts: HomeCounts;
}

export interface VideoRoot {
	id: number;
	name: string;
	item_count: number;
	cover_item_id: number | null;
}

export interface VideoRootsResponse {
	roots: VideoRoot[];
}

export interface VideoBreadcrumb {
	name: string;
	path: string;
}

export interface VideoFolder {
	name: string;
	path: string;
	item_count: number;
	cover_item_id: number | null;
}

export interface VideoLibraryRef {
	id: number;
	name: string;
}

export interface VideoRootRef {
	id: number;
	name: string;
}

export interface VideoBrowseResponse {
	library: VideoLibraryRef;
	root: VideoRootRef;
	path: string;
	parent: string | null;
	crumbs: VideoBreadcrumb[];
	folders: VideoFolder[];
	items: MediaItem[];
	total: number;
	limit: number;
	offset: number;
}

export type VideoBrowseResult = VideoRootsResponse | VideoBrowseResponse;

export interface VideoBrowseQuery {
	root?: number;
	path?: string;
	limit?: number;
	offset?: number;
}

export interface MusicArtist {
	id: number;
	name: string;
	album_count: number;
	track_count: number;
}

export interface MusicArtistDetail extends MusicArtist {
	albums: MusicAlbum[];
	total?: number;
	limit?: number;
	offset?: number;
}

export type MusicAlbumSort = 'title' | 'artist' | 'year' | 'added';

export interface MusicAlbum {
	id: number;
	title: string;
	artist: string;
	artist_id: number;
	year: number | null;
	track_count: number;
	duration_seconds: number;
	has_cover: boolean;
}

export interface MusicTrack {
	id: number;
	title: string;
	track_no: number | null;
	disc_no: number;
	artist: string;
	duration_seconds: number | null;
	codec: string | null;
	bitrate: number | null;
	playable: boolean;
	// La búsqueda de pistas puede incluir el álbum; el detalle de álbum ya lo
	// conoce por contexto y no necesita repetirlo.
	album?: string;
	album_id?: number;
}

export interface MusicAlbumDetail extends MusicAlbum {
	tracks: MusicTrack[];
	total?: number;
	limit?: number;
	offset?: number;
}

export interface MusicArtistsResponse {
	artists: MusicArtist[];
	total: number;
	limit?: number;
	offset?: number;
}

export interface MusicAlbumsResponse {
	albums: MusicAlbum[];
	total: number;
	limit?: number;
	offset?: number;
}

export interface MusicTracksResponse {
	tracks: MusicTrack[];
	total: number;
	limit?: number;
	offset?: number;
}

export interface MusicPlayResponse {
	url: string;
	mime: string;
	expires_at: string;
	playable: boolean;
}

export interface MusicArtistsQuery {
	limit?: number;
	offset?: number;
	q?: string;
	letter?: string;
}

export interface MusicAlbumsQuery {
	limit?: number;
	offset?: number;
	q?: string;
	letter?: string;
	sort?: MusicAlbumSort;
	artist?: number;
}

export interface MusicTracksQuery {
	limit?: number;
	offset?: number;
	q?: string;
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
	method: 'direct' | 'copy' | 'transcode' | '';
	hardware: boolean;
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
	path?: string;
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
	rating?: number | null;
	trailer_url?: string | null;
	trailer_language?: 'es' | 'en' | null;
	match_status: 'auto' | 'manual' | 'none' | null;
	favorite?: boolean;
	watchlist?: boolean;
}

export type UserListName = 'favorite' | 'watchlist';
export type UserListKind = 'item' | 'series' | 'collection';
export type ListsBlockKind = 'items' | 'series' | 'collections';

export interface ListTotals {
	items: number;
	series: number;
	collections: number;
}

export interface HomeListEntry {
	kind: 'item' | 'series';
	item?: MediaItem;
	series?: Series;
}

export interface ListsResponse {
	items: MediaItem[];
	series: Series[];
	collections: Collection[];
	totals: ListTotals;
	total?: number;
	limit: number;
	offset: number;
}

export interface ListsQuery {
	kind?: ListsBlockKind;
	limit?: number;
	offset?: number;
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
	folder?: string;
	original_title: string | null;
	year: number | null;
	tagline: string | null;
	overview: string | null;
	overview_language: string | null;
	poster_url: string | null;
	backdrop_url: string | null;
	genres: string[];
	rating: number | null;
	trailer_url?: string | null;
	trailer_language?: 'es' | 'en' | null;
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
	favorite?: boolean;
	watchlist?: boolean;
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
	caps?: string;
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

export type SettingType = 'bool' | 'int' | 'float' | 'string' | 'duration' | 'enum' | 'json';

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

export type PublicSettingsResponse = Record<string, unknown>;

export type SettingsValues = Record<string, string | boolean>;
export type SettingsChanges = Record<string, unknown>;

export interface ScanStarted {
	started: boolean;
}

export interface ServerVersion {
	server: string;
	commit: string;
	built_at: string;
	api_level?: number;
	min_client?: string;
}

export interface CurrentUserResponse {
	username: string;
	role: UserRole;
	all_libraries?: boolean;
	library_ids?: number[];
}

export type UserRole = 'admin' | 'user';

export interface ManagedUser {
	id: number;
	username: string;
	role: UserRole;
	disabled: boolean;
	all_libraries: boolean;
	library_ids: number[];
	created_at: string;
	active_sessions: number;
}

export interface UsersResponse {
	users: ManagedUser[];
}

export interface ActivityEntry {
	user_id: number;
	username: string;
	item_id: number;
	title: string;
	kind: MediaItem['kind'];
	series_title: string | null;
	season: number | null;
	episode: number | null;
	position_seconds: number;
	duration_seconds: number;
	watched: boolean;
	updated_at: string;
}

export interface ActivityResponse {
	entries: ActivityEntry[];
}

export interface ActivityQuery {
	limit?: number;
	user_id?: number;
}

export interface CreateUserInput {
	username: string;
	password: string;
	role: UserRole;
	all_libraries: boolean;
	library_ids: number[];
}

export interface UpdateUserInput {
	role?: UserRole;
	disabled?: boolean;
	all_libraries?: boolean;
}

export interface UpdateStatus {
	phase: string;
	tag: string;
	message: string;
	started_at: string;
	finished_at: string;
	result: 'ok' | 'failed' | 'rolled_back' | '';
}

export interface UpdateInfo {
	current: string;
	latest: string;
	available: boolean;
	checked_at: string;
	notes: string;
	apply_supported: boolean;
	check_enabled: boolean;
	status: UpdateStatus;
}

export interface Backup {
	name: string;
	size_bytes: number;
	created_at: string;
}

export interface BackupsResponse {
	enabled: boolean;
	retention_days: number;
	backups: Backup[];
	last_error: string;
}

export interface TrashItem {
	id: number;
	library_id: number;
	title: string;
	year: number | null;
	kind: string;
	original_path: string;
	moved_kind: 'file' | 'folder';
	deleted_by: string | null;
	deleted_at: string;
	expires_at: string;
}

export interface TrashResponse {
	items: TrashItem[];
}

export interface TrashRestoreResponse {
	item_id: number;
}

export interface TrashPurgeResponse {
	purged: number;
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
	headers?: Record<string, string>;
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

	async login(
		credentials: LoginRequest,
		clientIP = '',
		userAgent = ''
	): Promise<LoginResponse> {
		const headers: Record<string, string> = {};
		const ip = clientIP.trim();
		if (ip) {
			headers['X-Forwarded-For'] = ip;
		}
		const agent = sanitizeUserAgent(userAgent);
		if (agent) {
			headers['User-Agent'] = agent;
		}
		return this.request<LoginResponse>('/api/v1/auth/login', {
			method: 'POST',
			body: credentials,
			headers: Object.keys(headers).length > 0 ? headers : undefined
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

	async getSessions(token: string): Promise<SessionsResponse> {
		return this.get<SessionsResponse>('/api/v1/auth/sessions', token);
	}

	async closeSession(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/auth/sessions/${id}`, token);
	}

	async approvePairing(code: string, confirm: boolean, token: string): Promise<PairPreview> {
		return this.post<PairPreview>('/api/v1/auth/pair/approve', { code, confirm }, token);
	}

	async getUserSessions(userId: number, token: string): Promise<SessionsResponse> {
		return this.get<SessionsResponse>(`/api/v1/users/${userId}/sessions`, token);
	}

	async closeUserSession(userId: number, sessionId: number, token: string): Promise<void> {
		await this.delete(`/api/v1/users/${userId}/sessions/${sessionId}`, token);
	}

	async listUsers(token: string): Promise<UsersResponse> {
		return this.get<UsersResponse>('/api/v1/users', token);
	}

	async getActivity(query: ActivityQuery, token: string): Promise<ActivityResponse> {
		const params = new URLSearchParams();
		if (query.limit !== undefined) {
			params.set('limit', String(query.limit));
		}
		if (query.user_id !== undefined) {
			params.set('user_id', String(query.user_id));
		}
		const suffix = params.toString() ? `?${params}` : '';
		return this.get<ActivityResponse>(`/api/v1/activity${suffix}`, token);
	}

	async createUser(input: CreateUserInput, token: string): Promise<ManagedUser> {
		return this.post<ManagedUser>('/api/v1/users', input, token);
	}

	async updateUser(id: number, changes: UpdateUserInput, token: string): Promise<ManagedUser> {
		return this.request<ManagedUser>(`/api/v1/users/${id}`, {
			method: 'PATCH',
			body: changes,
			token
		});
	}

	async setUserLibraries(
		id: number,
		libraryIDs: number[],
		token: string
	): Promise<ManagedUser> {
		return this.put<ManagedUser>(`/api/v1/users/${id}/libraries`, { library_ids: libraryIDs }, token);
	}

	async resetUserPassword(id: number, password: string, token: string): Promise<void> {
		await this.post<void>(`/api/v1/users/${id}/password`, { password }, token);
	}

	async deleteUser(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/users/${id}`, token);
	}

	async listLibraries(token: string): Promise<LibrariesResponse> {
		return this.get<LibrariesResponse>('/api/v1/libraries', token);
	}

	async getUploadTargets(token: string): Promise<UploadTargetsResponse> {
		return this.get<UploadTargetsResponse>('/api/v1/uploads/targets', token);
	}

	async createLibrary(input: CreateLibraryInput, token: string): Promise<Library> {
		return this.post<Library>('/api/v1/libraries', input, token);
	}

	async renameLibrary(id: number, name: string, token: string): Promise<Library> {
		return this.request<Library>(`/api/v1/libraries/${id}`, {
			method: 'PATCH',
			body: { name },
			token
		});
	}

	async addLibraryPath(id: number, path: string, token: string): Promise<Library> {
		return this.post<Library>(`/api/v1/libraries/${id}/paths`, { path }, token);
	}

	async removeLibraryPath(id: number, pathId: number, token: string): Promise<void> {
		await this.delete(`/api/v1/libraries/${id}/paths/${pathId}`, token);
	}

	async deleteLibrary(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/libraries/${id}`, token);
	}

	async browseFS(path: string | null | undefined, token: string): Promise<FSResponse> {
		const suffix = path ? `?path=${encodeURIComponent(path)}` : '';
		return this.get<FSResponse>(`/api/v1/fs${suffix}`, token);
	}

	async browseVideoLibrary(
		id: number,
		query: VideoBrowseQuery,
		token: string
	): Promise<VideoBrowseResult> {
		const params = new URLSearchParams();
		if (query.root !== undefined) {
			params.set('root', String(query.root));
		}
		if (query.path) {
			params.set('path', query.path);
		}
		if (query.limit !== undefined) {
			params.set('limit', String(query.limit));
		}
		if (query.offset !== undefined) {
			params.set('offset', String(query.offset));
		}
		const suffix = params.toString() ? `?${params}` : '';
		return this.get<VideoBrowseResult>(`/api/v1/libraries/${id}/browse${suffix}`, token);
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

	async getUpdate(token: string): Promise<UpdateInfo> {
		return this.get<UpdateInfo>('/api/v1/system/update', token);
	}

	async checkUpdate(token: string): Promise<UpdateInfo> {
		return this.post<UpdateInfo>('/api/v1/system/update/check', undefined, token);
	}

	async applyUpdate(tag: string, token: string): Promise<UpdateInfo> {
		return this.post<UpdateInfo>('/api/v1/system/update/apply', { tag }, token);
	}

	async getBackups(token: string): Promise<BackupsResponse> {
		return this.get<BackupsResponse>('/api/v1/system/backups', token);
	}

	async createBackup(token: string): Promise<Backup> {
		return this.post<Backup>('/api/v1/system/backups', undefined, token);
	}

	async getSettings(token: string): Promise<SettingsResponse> {
		return this.get<SettingsResponse>('/api/v1/settings', token);
	}

	async getPublicSettings(token: string): Promise<PublicSettingsResponse> {
		return this.get<PublicSettingsResponse>('/api/v1/settings/public', token);
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

	async getPerson(id: number, token: string): Promise<PersonResponse> {
		return this.get<PersonResponse>(`/api/v1/people/${id}`, token);
	}

	async trashItem(id: number, token: string): Promise<TrashItem> {
		return this.request<TrashItem>(`/api/v1/items/${id}`, {
			method: 'DELETE',
			token
		});
	}

	async listTrash(token: string): Promise<TrashResponse> {
		return this.get<TrashResponse>('/api/v1/trash', token);
	}

	async restoreTrash(id: number, token: string): Promise<TrashRestoreResponse> {
		return this.post<TrashRestoreResponse>(`/api/v1/trash/${id}/restore`, undefined, token);
	}

	async deleteTrash(id: number, token: string): Promise<void> {
		await this.delete(`/api/v1/trash/${id}`, token);
	}

	async purgeTrash(token: string): Promise<TrashPurgeResponse> {
		return this.post<TrashPurgeResponse>('/api/v1/trash/purge', undefined, token);
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

	async getLists(list: UserListName, query: ListsQuery, token: string): Promise<ListsResponse> {
		const params = new URLSearchParams();
		if (query.kind) {
			params.set('kind', query.kind);
		}
		if (query.limit !== undefined) {
			params.set('limit', String(query.limit));
		}
		if (query.offset !== undefined) {
			params.set('offset', String(query.offset));
		}
		const suffix = params.toString() ? `?${params}` : '';
		return this.get<ListsResponse>(`/api/v1/lists/${list}${suffix}`, token);
	}

	async setItemList(id: number, list: UserListName, on: boolean, token: string): Promise<void> {
		await this.setList(`/api/v1/items/${id}/lists/${list}`, on, token);
	}

	async setSeriesList(id: number, list: UserListName, on: boolean, token: string): Promise<void> {
		await this.setList(`/api/v1/series/${id}/lists/${list}`, on, token);
	}

	async setCollectionList(id: number, list: UserListName, on: boolean, token: string): Promise<void> {
		await this.setList(`/api/v1/collections/${id}/lists/${list}`, on, token);
	}

	private async setList(path: string, on: boolean, token: string): Promise<void> {
		if (on) {
			await this.put<void>(path, undefined, token);
			return;
		}
		await this.delete(path, token);
	}

	async getMusicArtists(query: MusicArtistsQuery, token: string): Promise<MusicArtistsResponse> {
		const params = new URLSearchParams();
		appendPagination(params, query);
		if (query.q) {
			params.set('q', query.q);
		}
		if (query.letter) {
			params.set('letter', query.letter);
		}
		return this.get<MusicArtistsResponse>(`/api/v1/music/artists${querySuffix(params)}`, token);
	}

	async getMusicArtist(
		id: number,
		query: { limit?: number; offset?: number } = {},
		token: string
	): Promise<MusicArtistDetail> {
		const params = new URLSearchParams();
		appendPagination(params, query);
		return this.get<MusicArtistDetail>(
			`/api/v1/music/artists/${id}${querySuffix(params)}`,
			token
		);
	}

	async getMusicAlbums(query: MusicAlbumsQuery, token: string): Promise<MusicAlbumsResponse> {
		const params = new URLSearchParams();
		appendPagination(params, query);
		if (query.q) {
			params.set('q', query.q);
		}
		if (query.letter) {
			params.set('letter', query.letter);
		}
		if (query.sort) {
			params.set('sort', query.sort);
		}
		if (query.artist !== undefined) {
			params.set('artist', String(query.artist));
		}
		return this.get<MusicAlbumsResponse>(`/api/v1/music/albums${querySuffix(params)}`, token);
	}

	async getMusicAlbum(
		id: number,
		query: { limit?: number; offset?: number } = {},
		token: string
	): Promise<MusicAlbumDetail> {
		const params = new URLSearchParams();
		appendPagination(params, query);
		return this.get<MusicAlbumDetail>(`/api/v1/music/albums/${id}${querySuffix(params)}`, token);
	}

	async getMusicTracks(query: MusicTracksQuery, token: string): Promise<MusicTracksResponse> {
		const params = new URLSearchParams();
		appendPagination(params, query);
		if (query.q) {
			params.set('q', query.q);
		}
		return this.get<MusicTracksResponse>(`/api/v1/music/tracks${querySuffix(params)}`, token);
	}

	async getMusicPlay(id: number, token: string): Promise<MusicPlayResponse> {
		return this.get<MusicPlayResponse>(`/api/v1/music/tracks/${id}/play`, token);
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
		if (query.caps) {
			params.set('caps', query.caps);
		}
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

	async getFileInfo(id: number, token: string, caps = ''): Promise<FileInfo> {
		const params = new URLSearchParams();
		if (caps) {
			params.set('caps', caps);
		}
		const suffix = params.toString() ? `?${params}` : '';
		return this.get<FileInfo>(`/api/v1/items/${id}/file-info${suffix}`, token);
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

	async getCollectionCandidates(
		id: number,
		query: string,
		token: string
	): Promise<CollectionCandidatesResponse> {
		const clean = query.trim();
		const suffix = clean ? `?${new URLSearchParams({ q: clean })}` : '';
		return this.get<CollectionCandidatesResponse>(
			`/api/v1/collections/${id}/candidates${suffix}`,
			token
		);
	}

	async matchCollection(
		id: number,
		tmdbCollectionID: number,
		token: string
	): Promise<CollectionDetail> {
		return this.put<CollectionDetail>(
			`/api/v1/collections/${id}/match`,
			{ tmdb_collection_id: tmdbCollectionID },
			token
		);
	}

	async resetCollectionMatch(id: number, token: string): Promise<CollectionDetail> {
		return this.request<CollectionDetail>(`/api/v1/collections/${id}/match`, {
			method: 'DELETE',
			token
		});
	}

	async getAllCollections(token: string): Promise<Collection[]> {
		const response = await this.get<CollectionsResponse>('/api/v1/collections?all=1', token);
		return response.collections ?? [];
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
		const headers: Record<string, string> = {
			Accept: 'application/json',
			...(options.headers ?? {})
		};
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

function appendPagination(
	params: URLSearchParams,
	query: { limit?: number; offset?: number }
): void {
	if (query.limit !== undefined) {
		params.set('limit', String(query.limit));
	}
	if (query.offset !== undefined) {
		params.set('offset', String(query.offset));
	}
}

function querySuffix(params: URLSearchParams): string {
	return params.toString() ? `?${params}` : '';
}

function sanitizeUserAgent(value: string): string {
	const clean = value.replace(/[\r\n\u2028\u2029]+/g, ' ').trim();
	return Array.from(clean).slice(0, 300).join('');
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
