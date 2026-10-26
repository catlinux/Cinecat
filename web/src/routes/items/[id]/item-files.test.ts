import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { FileInfo, FileInfoResult, ImageOptions, MediaItem } from '$lib/api';
import FilePanel from '$lib/components/FilePanel.svelte';
import ItemPage from './+page.svelte';
import { actions, load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function makeItem(overrides: Partial<MediaItem> = {}): MediaItem {
	return {
		id: 7,
		library_id: 1,
		path: '/media/peliculas/El Padrino (1972).mkv',
		title: 'El Padrino',
		file_title: 'El Padrino (1972)',
		year: 1972,
		kind: 'movie',
		season: null,
		episode: null,
		tmdb_id: 238,
		overview: 'Una familia muy unida.',
		poster_url: 'https://images.example/poster.jpg',
		poster_custom: false,
		original_title: 'The Godfather',
		imdb_id: 'tt0068646',
		tagline: null,
		genres: ['Drama'],
		runtime_minutes: 112,
		rating: 7.8,
		directors: ['Francis Ford Coppola'],
		cast: ['Marlon Brando'],
		backdrop_url: 'https://images.example/backdrop.jpg',
		backdrop_custom: false,
		overview_language: 'es-ES',
		match_status: 'auto',
		match_score: 1,
		watched: false,
		...overrides
	};
}

function makeFileInfo(overrides: Partial<FileInfo> = {}): FileInfo {
	return {
		container: 'matroska',
		container_long_name: 'Matroska / WebM',
		size_bytes: 1500000000,
		duration_seconds: 6732,
		bit_rate: 1371742,
		modified_at: '2026-10-26T10:00:00Z',
		file_name: 'El Padrino.mkv',
		path: 'El Padrino.mkv',
		video: [
			{
				index: 0,
				codec: 'h264',
				codec_long_name: 'H.264 / AVC / MPEG-4 AVC',
				profile: 'High',
				width: 1920,
				height: 1080,
				display_aspect_ratio: '16:9',
				frame_rate: 23.976,
				bit_rate: 5000000,
				pixel_format: 'yuv420p',
				bit_depth: 8,
				hdr: '',
				language: 'spa',
				title: '',
				default: true
			}
		],
		audio: [
			{
				index: 1,
				codec: 'ac3',
				codec_long_name: 'ATSC A/52A (AC-3)',
				profile: '',
				channels: 6,
				channel_layout: '5.1(side)',
				sample_rate: 48000,
				bit_rate: 448000,
				language: 'spa',
				title: 'Castellano',
				default: true
			},
			{
				index: 2,
				codec: 'aac',
				codec_long_name: 'AAC',
				profile: 'LC',
				channels: 2,
				channel_layout: 'stereo',
				sample_rate: 48000,
				bit_rate: 192000,
				language: 'eng',
				title: 'English',
				default: false
			}
		],
		subtitles: [
			{
				index: 3,
				codec: 'subrip',
				language: 'spa',
				title: '',
				forced: true,
				default: false,
				text: true
			},
			{
				index: 4,
				codec: 'hdmv_pgs_subtitle',
				language: 'eng',
				title: '',
				forced: false,
				default: false,
				text: false
			}
		],
		chapters: 0,
		quality: 'Full HD',
		playback: {
			mode: 'hls',
			description: 'Se recodifica a H.264 + AAC estéreo por HLS',
			supported: true
		},
		...overrides
	};
}

function fileInfoResult(info: FileInfo = makeFileInfo()): FileInfoResult {
	return { data: info, error: null };
}

function loadEvent(fetchImpl: typeof fetch, isAdmin = false) {
	return {
		params: { id: '7' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL('http://localhost/items/7'),
		parent: async () => ({ isAdmin })
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	action: string,
	fields: Record<string, string> = {},
	token = 'token-123'
) {
	return {
		request: new Request(`http://localhost/items/7?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		params: { id: '7' },
		url: new URL('http://localhost/items/7')
	};
}

describe('carga del panel de archivo', () => {
	it('devuelve la ficha y el file-info como promesa', async () => {
		const info = makeFileInfo();
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(makeItem());
			}
			if (url.pathname === '/api/v1/items/7/file-info') {
				return jsonResponse(info);
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock) as never)) as {
			item: MediaItem;
			fileInfo: Promise<FileInfoResult>;
		};
		expect(result.item.title).toBe('El Padrino');
		const fileInfo = await result.fileInfo;
		expect(fileInfo.error).toBeUndefined();
		expect(fileInfo.data?.container).toBe('matroska');
	});

	for (const [status, error] of [
		[404, 'not_found'],
		[502, 'failed'],
		[503, 'unavailable']
	] as const) {
		it(`convierte el error ${status} del archivo en ${error}`, async () => {
			const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
				const url = new URL(String(input));
				if (url.pathname === '/api/v1/items/7') {
					return jsonResponse(makeItem());
				}
				if (url.pathname === '/api/v1/items/7/file-info') {
					return jsonResponse({ error: 'fallo de archivo' }, status);
				}
				throw new Error(`URL inesperada: ${url}`);
			});

			const result = (await load(loadEvent(fetchMock) as never)) as {
				item: MediaItem;
				fileInfo: Promise<FileInfoResult>;
			};
			const fileInfo = await result.fileInfo;
			expect(result.item.title).toBe('El Padrino');
			expect(fileInfo).toEqual({ error });
		});
	}
});

function renderFilePanel(result: FileInfoResult): string {
	return render(FilePanel, { props: { result } as never }).body;
}

describe('FilePanel', () => {
	it('muestra la información de un MKV con varias pistas', () => {
		const body = renderFilePanel(fileInfoResult());

		expect(body).toMatch(/<details[^>]*\bopen\b/);
		expect(body).toContain('Full HD');
		expect(body).toContain('Matroska / WebM');
		expect(body).toContain('1,4 GB');
		expect(body).toContain('1 h 52 min');
		expect(body).toContain('1,4 Mb/s');
		expect(body).toContain('El Padrino.mkv');

		expect(body).toContain('Vídeo');
		expect(body).toContain('h264');
		expect(body).toContain('H.264 / AVC / MPEG-4 AVC');
		expect(body).toContain('High');
		expect(body).toContain('1920 × 1080 (16:9)');
		expect(body).toContain('23,976');
		expect(body).toContain('8 bits');
		expect(body).toContain('5 Mb/s');

		expect(body).toContain('Audio');
		expect(body).toContain('5.1');
		expect(body).toContain('estéreo');
		expect(body).toContain('48 kHz');
		expect(body).toContain('448 kb/s');
		expect(body).toContain('Español');
		expect(body).toContain('Inglés');
		expect(body).toContain('Por defecto');

		expect(body).toContain('Subtítulos');
		expect(body).toContain('subrip');
		expect(body).toContain('hdmv_pgs_subtitle');
		expect(body).toContain('forzado');
		expect(body).toContain('imagen');
		expect(body).toContain('Se recodifica a H.264 + AAC estéreo por HLS');
	});

	it('muestra un AVI antiguo sin subtítulos', () => {
		const body = renderFilePanel(
			fileInfoResult(
				makeFileInfo({
					container: 'avi',
					container_long_name: 'AVI',
					size_bytes: 734003200,
					duration_seconds: 5400,
					bit_rate: 1088000,
					quality: 'SD',
					video: [
						{
							index: 0,
							codec: 'mpeg4',
							codec_long_name: 'MPEG-4 part 2',
							profile: '',
							width: 720,
							height: 576,
							display_aspect_ratio: '5:4',
							frame_rate: 25,
							bit_rate: 900000,
							pixel_format: 'yuv420p',
							bit_depth: 8,
							hdr: '',
							language: '',
							title: '',
							default: true
						}
					],
					audio: [],
					subtitles: [],
					playback: { mode: 'direct', description: 'Reproducción directa (AVI)', supported: true }
				})
			)
		);

		expect(body).toContain('SD');
		expect(body).toContain('700 MB');
		expect(body).toContain('720 × 576 (5:4)');
		expect(body).toContain('Sin pistas de audio.');
		expect(body).toContain('Sin subtítulos incrustados');
	});

	it('muestra HDR y profundidad de color', () => {
		const body = renderFilePanel(
			fileInfoResult(
				makeFileInfo({
					quality: '4K',
					video: [
						{
							index: 0,
							codec: 'hevc',
							codec_long_name: 'H.265 / HEVC',
							profile: 'Main 10',
							width: 3840,
							height: 2160,
							display_aspect_ratio: '16:9',
							frame_rate: 23.976,
							bit_rate: 15000000,
							pixel_format: 'yuv420p10le',
							bit_depth: 10,
							hdr: 'HDR10',
							language: '',
							title: '',
							default: true
						}
					]
				})
			)
		);

		expect(body).toContain('4K');
		expect(body).toContain('HDR10');
		expect(body).toContain('10 bits');
		expect(body).toContain('15 Mb/s');
	});

	it('destaca los contenedores no soportados para reproducción', () => {
		const body = renderFilePanel(
			fileInfoResult(
				makeFileInfo({
					playback: {
						mode: '',
						description: 'Contenedor no soportado para reproducción',
						supported: false
					}
				})
			)
		);
		expect(body).toContain('file-playback--warning');
		expect(body).toContain('Contenedor no soportado para reproducción');
	});

	for (const [error, message] of [
		['not_found', 'No se encuentra el archivo en el servidor'],
		['unavailable', 'ffprobe no está disponible en el servidor'],
		['failed', 'No se pudo analizar el archivo']
	] as const) {
		it(`muestra el aviso controlado para ${error}`, () => {
			const body = renderFilePanel({ data: null, error });
			expect(body).toContain(message);
			expect(body).toContain('class="error file-panel-error"');
		});
	}
});

interface PageOptions {
	isAdmin?: boolean;
	imagenes?: boolean;
	form?: unknown;
	fileInfo?: FileInfoResult;
}

function renderPage(item: MediaItem, options: PageOptions = {}): string {
	return render(ItemPage, {
		props: {
			data: {
				item,
				isAdmin: options.isAdmin ?? false,
				corregir: false,
				imagenes: options.imagenes ?? false,
				fileInfo: options.fileInfo ?? fileInfoResult()
			},
			form: options.form ?? null
		} as never
	}).body;
}

describe('ImagePanel', () => {
	it('solo aparece para administradores en películas', () => {
		const admin = renderPage(makeItem({ poster_custom: true }), { isAdmin: true, imagenes: true });
		expect(admin).toContain('Cambiar imagen');
		expect(admin).toContain('Cartel');
		expect(admin).toContain('Fondo');
		expect(admin).toContain('personalizada');
		expect(admin).toContain('Restaurar la de TMDB');
		expect(admin).toContain('Usar esta URL');
		expect(admin).toContain('Ver las que ofrece TMDB');

		const user = renderPage(makeItem(), { isAdmin: false });
		expect(user).not.toContain('Cambiar imagen');

		const episode = renderPage(makeItem({ kind: 'episode' }), {
			isAdmin: true,
			imagenes: true
		});
		expect(episode).not.toContain('Cambiar imagen');
	});

	it('no ofrece restaurar si la imagen no es personalizada', () => {
		const body = renderPage(makeItem({ poster_custom: false }), {
			isAdmin: true,
			imagenes: true
		});
		expect(body).not.toContain('personalizada');
		expect(body).not.toContain('Restaurar la de TMDB');
	});

	it('muestra la cuadrícula de imágenes de TMDB', () => {
		const imageOptions: ImageOptions = {
			posters: [
				{
					url: 'https://images.example/w342/poster.jpg',
					full_url: 'https://images.example/w780/poster.jpg',
					language: 'es',
					vote_average: 7.8,
					width: 342,
					height: 513
				}
			],
			backdrops: []
		};
		const body = renderPage(makeItem(), {
			isAdmin: true,
			form: { imagePanel: true, imageKind: 'poster', imageOptions }
		});

		expect(body).toContain('https://images.example/w342/poster.jpg');
		expect(body).toContain('https://images.example/w780/poster.jpg');
		expect(body).toContain('Español');
		expect(body).toContain('342 × 513');
		expect(body).toContain('Usar esta');
		expect(body).not.toContain('Sin imágenes disponibles.');
	});
});

describe('acciones del panel de imagen', () => {
	it('fija una URL válida con PUT', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(makeItem()));

		const result = (await actions.imageSet!(
			actionEvent(fetchMock, 'imageSet', {
				kind: 'poster',
				url: 'https://images.example/poster.jpg'
			}) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Cartel actualizado');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/images/poster');
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body)).toEqual({ url: 'https://images.example/poster.jpg' });
	});

	it('rechaza una URL vacía o que no sea http(s)', async () => {
		for (const url of ['', 'ftp://images.example/p.jpg', 'images.example/p.jpg']) {
			const fetchMock = vi.fn();
			const result = (await actions.imageSet!(
				actionEvent(fetchMock, 'imageSet', { kind: 'poster', url }) as never
			)) as { status: number; data: { message: string } };
			expect(result.status).toBe(400);
			expect(result.data.message).toBe('URL no válida');
			expect(fetchMock).not.toHaveBeenCalled();
		}
	});

	it('restaura una imagen con DELETE', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(makeItem()));

		const result = (await actions.imageReset!(
			actionEvent(fetchMock, 'imageReset', { kind: 'backdrop' }) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Fondo restaurado');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/images/backdrop');
		expect(init.method).toBe('DELETE');
	});

	it('carga las imágenes de TMDB con la action imageList', async () => {
		const options: ImageOptions = {
			posters: [
				{
					url: 'https://images.example/w342/poster.jpg',
					full_url: 'https://images.example/w780/poster.jpg',
					language: 'es',
					vote_average: 7,
					width: 342,
					height: 513
				}
			],
			backdrops: []
		};
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(options));

		const result = (await actions.imageList!(
			actionEvent(fetchMock, 'imageList', { kind: 'poster' }) as never
		)) as {
			imageKind: string;
			imageOptions: ImageOptions;
			imagePanel: boolean;
		};

		expect(result.imageKind).toBe('poster');
		expect(result.imagePanel).toBe(true);
		expect(result.imageOptions.posters).toHaveLength(1);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/images');
		expect(init.method).toBe('GET');
	});

	it('traduce los errores de imageList y el 403 de permisos', async () => {
		const cases = [
			[409, 'Identifica primero la película para ver las imágenes de TMDB'],
			[502, 'No se pudo consultar TMDB'],
			[503, 'TMDB no está configurado en el servidor'],
			[403, 'Solo los administradores pueden cambiar las imágenes.']
		] as const;

		for (const [status, message] of cases) {
			const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, status));
			const result = (await actions.imageList!(
				actionEvent(fetchMock, 'imageList', { kind: 'poster' }) as never
			)) as { status: number; data: { message: string } };
			expect(result.status).toBe(status);
			expect(result.data.message).toBe(message);
		}
	});
});
