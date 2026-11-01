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
		progress: null,
		collections: [],
		series_id: null,
		series_title: null,
		episode_title: null,
		still_url: null,
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
			method: 'transcode',
			hardware: false,
			description: 'Se recodifica a H.264 + AAC estéreo por HLS',
			supported: true
		},
		...overrides
	};
}

function fileInfoResult(info: FileInfo = makeFileInfo()): FileInfoResult {
	return { data: info, error: null };
}

function loadEvent(fetchImpl: typeof fetch, isAdmin = false, caps = '') {
	return {
		params: { id: '7' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL('http://localhost/items/7'),
		parent: async () => ({ isAdmin }),
		cookies: { get: () => caps }
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

function subtitleUploadEvent(
	fetchImpl: typeof fetch,
	file: File,
	language = '',
	label = ''
) {
	const form = new FormData();
	form.append('file', file);
	if (language) {
		form.append('language', language);
	}
	if (label) {
		form.append('label', label);
	}
	return {
		request: new Request('http://localhost/items/7?/subtitleUpload', {
			method: 'POST',
			body: form
		}),
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		params: { id: '7' },
		url: new URL('http://localhost/items/7')
	};
}

describe('carga del panel de archivo', () => {
	it('pasa las capacidades saneadas al file-info y a las opciones', async () => {
		const calls: string[] = [];
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(makeItem());
			}
			if (url.pathname === '/api/v1/items/7/file-info') {
				calls.push(url.toString());
				return jsonResponse(makeFileInfo());
			}
			if (url.pathname === '/api/v1/items/7/play') {
				calls.push(url.toString());
				return jsonResponse({ url: '', mode: 'direct' });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock, false, 'HEVC10, otro') as never)) as {
			fileInfo: Promise<FileInfoResult>;
			playbackOptions: Promise<unknown>;
		};
		await result.fileInfo;
		await result.playbackOptions;

		expect(calls).toHaveLength(2);
		for (const call of calls) {
			expect(new URL(call).searchParams.get('caps')).toBe('hevc,hevc10');
		}
	});

	it('separa los subtítulos subidos de las opciones de reproducción', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(makeItem());
			}
			if (url.pathname === '/api/v1/items/7/file-info') {
				return jsonResponse(makeFileInfo());
			}
			if (url.pathname === '/api/v1/items/7/play') {
				return jsonResponse({
					url: '',
					mode: 'direct',
					subtitles: [
						{
							id: 'u1',
							source: 'uploaded',
							language: 'es',
							language_name: 'Español',
							title: 'Castellano',
							codec: 'subrip',
							forced: false,
							default: false,
							url: '/sub/1'
						},
						{
							id: 'x0',
							source: 'external',
							language: 'en',
							language_name: 'Inglés',
							title: '',
							codec: 'subrip',
							forced: false,
							default: false,
							url: '/sub/2'
						}
					]
				});
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock, true) as never)) as {
			uploadedSubtitles: { id: string }[];
		};
		expect(Array.isArray(result.uploadedSubtitles)).toBe(true);
		expect(result.uploadedSubtitles.map((item) => item.id)).toEqual(['u1']);
	});

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
	it('muestra la gestión de subtítulos subidos solo a administradores', () => {
		const uploaded = [
			{
				id: 'u1',
				source: 'uploaded' as const,
				language: 'es',
				language_name: 'Español',
				title: 'Castellano',
				codec: 'subrip',
				forced: false,
				default: false,
				url: '/api/v1/stream/7/subtitles/u1.vtt?exp=1&sig=abc'
			}
		];
		const admin = render(FilePanel, {
			props: {
				result: fileInfoResult(),
				itemId: 7,
				isAdmin: true,
				uploadedSubtitles: uploaded,
				subtitleMessage: 'Subtítulo subido.',
				subtitleSuccess: true
			} as never
		}).body;
		expect(admin).toContain('Subidos');
		expect(admin).toContain('Español · Castellano');
		expect(admin).toContain('u1');
		expect(admin).toContain('Eliminar');
		expect(admin).toContain('Subir subtítulo');
		expect(admin).toContain('name="file"');
		expect(admin).toContain('accept=".srt,.vtt,.ass"');
		expect(admin).toContain('action="?/subtitleUpload"');
		expect(admin).toContain('action="?/subtitleDelete"');
		expect(admin).toContain('Subtítulo subido.');

		const user = render(FilePanel, {
			props: {
				result: fileInfoResult(),
				itemId: 7,
				isAdmin: false,
				uploadedSubtitles: uploaded
			} as never
		}).body;
		expect(user).not.toContain('Subir subtítulo');
	});

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

	it('oculta la ruta cuando la API no la envía', () => {
		const body = renderFilePanel(
			fileInfoResult(
				makeFileInfo({
					path: undefined
				})
			)
		);
		expect(body).not.toContain('<code>');
	});

	it('indica cuando la recodificación usa la gráfica', () => {
		const body = renderFilePanel(
			fileInfoResult(
				makeFileInfo({
					playback: {
						mode: 'hls',
						method: 'transcode',
						hardware: true,
						description: 'Se recodifica a H.264 + AAC estéreo por HLS',
						supported: true
					}
				})
			)
		);
		expect(body).toContain('Recodificación (gráfica)');
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
					playback: {
						mode: 'direct',
						method: 'direct',
						hardware: false,
						description: 'Reproducción directa (AVI)',
						supported: true
					}
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
						method: '',
						hardware: false,
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

describe('acciones de subtítulos', () => {
	it('sube un archivo como multipart', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ id: 'u1', language: 'es', label: 'Castellano', codec: 'subrip' }, 201)
		);
		const file = new File(['1\n00:00:01,000 --> 00:00:02,000\nHola\n'], 'Pelicula.srt', {
			type: 'text/plain'
		});

		const result = (await actions.subtitleUpload!(
			subtitleUploadEvent(fetchMock, file, 'es', 'Castellano') as never
		)) as { success?: boolean; subtitlePanel?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.subtitlePanel).toBe(true);
		expect(result.message).toContain('Subtítulo subido');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/subtitles');
		expect(init.method).toBe('POST');
		expect(init.headers['Content-Type']).toBeUndefined();
		expect(init.body).toBeInstanceOf(FormData);
		const form = init.body as FormData;
		expect(form.get('language')).toBe('es');
		expect(form.get('label')).toBe('Castellano');
		expect((form.get('file') as File).name).toBe('Pelicula.srt');
	});

	it('rechaza archivos no admitidos sin llamar a la API', async () => {
		const fetchMock = vi.fn();
		for (const [file, status, message] of [
			[new File(['x'], 'Pelicula.txt'), 400, 'Solo se admiten'],
			[new File(['x'], 'Pelicula.srt'), 400, 'idioma'],
			[
				new File([new Uint8Array(2 * 1024 * 1024 + 1)], 'Pelicula.srt'),
				413,
				'2 MB'
			]
		] as const) {
			const language = message === 'idioma' ? 'español' : '';
			const result = (await actions.subtitleUpload!(
				subtitleUploadEvent(fetchMock, file, language) as never
			)) as { status: number; data: { message: string } };
			expect(result.status).toBe(status);
			expect(result.data.message).toContain(message);
		}
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('elimina un subtítulo y traduce el 403', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const result = (await actions.subtitleDelete!(
			actionEvent(fetchMock, 'subtitleDelete', { subtitle_id: 'u1' }) as never
		)) as { success?: boolean; message?: string };
		expect(result.success).toBe(true);
		expect(result.message).toContain('Subtítulo eliminado');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/subtitles/u1');
		expect(init.method).toBe('DELETE');

		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'nope' }, 403));
		const denied = (await actions.subtitleDelete!(
			actionEvent(forbidden, 'subtitleDelete', { subtitle_id: 'u1' }) as never
		)) as { status: number; data: { message: string } };
		expect(denied.status).toBe(403);
		expect(denied.data.message).toContain('Solo los administradores');
	});
});
