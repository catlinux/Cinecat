import { describe, expect, it } from 'vitest';
import {
	UPLOAD_ACCEPT,
	buildEpisodeUploadName,
	buildUploadURL,
	formatUploadSize,
	isUploadableFileName,
	isUploadSubtitleName,
	isUploadVideoName,
	subtitleMatchesVideo,
	uploadErrorMessage,
	uploadFileStem,
	uploadSubdirForFile,
	validUploadComponent
} from './uploads';

describe('utilidades de subida', () => {
	it('reconoce vídeos y subtítulos por su extensión', () => {
		for (const name of ['Peli.MKV', 'clip.mp4', 'sub.srt', 'sub.EN.vtt', 'nota.ass', 'x.idx']) {
			expect(isUploadableFileName(name)).toBe(true);
		}
		for (const name of ['nota.txt', 'Peli', 'imagen.jpg', 'Peli.mkv.exe']) {
			expect(isUploadableFileName(name)).toBe(false);
		}
		expect(UPLOAD_ACCEPT).toContain('.mkv');
		expect(UPLOAD_ACCEPT).toContain('.srt');
	});

	it('formatea tamaños legibles', () => {
		expect(formatUploadSize(512)).toBe('512 B');
		expect(formatUploadSize(734003200)).toBe('700 MB');
	});

	it('construye la URL del proxy con los parámetros escapados', () => {
		const url = new URL(
			buildUploadURL({
				library: 4,
				dir: '/media/pelis con espacio',
				name: 'Peli (2020).mkv',
				subdir: 'Temporada 1'
			}),
			'http://localhost'
		);
		expect(url.pathname).toBe('/admin/libraries/upload');
		expect(url.searchParams.get('library')).toBe('4');
		expect(url.searchParams.get('dir')).toBe('/media/pelis con espacio');
		expect(url.searchParams.get('name')).toBe('Peli (2020).mkv');
		expect(url.searchParams.get('subdir')).toBe('Temporada 1');
	});

	it('traduce los errores de la API', () => {
		expect(uploadErrorMessage(403, { error: 'apagado' })).toContain('Permitir cambiar archivos');
		expect(uploadErrorMessage(403, null)).toContain('Administración → Biblioteca');
		expect(uploadErrorMessage(409, { error: 'ya existe' })).toBe('ya existe');
		expect(uploadErrorMessage(413, null)).toContain('tamaño máximo');
		expect(uploadErrorMessage(507, null)).toContain('espacio libre');
		expect(uploadErrorMessage(502, null)).toBe('No se pudo subir el archivo.');
	});
});

describe('nombre de archivo de episodio', () => {
	it('construye el nombre con temporada y episodio a dos cifras', () => {
		expect(buildEpisodeUploadName('The Big Bang Theory', 1, 5, 'Capitulo.MKV')).toBe(
			'The Big Bang Theory S01E05.mkv'
		);
		expect(buildEpisodeUploadName('Serie', 10, 123, 'Capitulo.mp4')).toBe(
			'Serie S10E123.mp4'
		);
	});

	it('conserva el idioma de los subtítulos', () => {
		expect(buildEpisodeUploadName('Serie', 2, 3, 'Capitulo.ES.SRT')).toBe(
			'Serie S02E03.es.srt'
		);
		expect(buildEpisodeUploadName('Serie', 2, 3, 'Capitulo.spa.ass')).toBe(
			'Serie S02E03.spa.ass'
		);
		expect(buildEpisodeUploadName('Serie', 2, 3, 'Capitulo.vtt')).toBe(
			'Serie S02E03.vtt'
		);
	});
});

describe('reparto de películas y subtítulos', () => {
	it('reconoce vídeos, subtítulos y nombres base', () => {
		expect(isUploadVideoName('Peli.MKV')).toBe(true);
		expect(isUploadSubtitleName('Peli.ES.SRT')).toBe(true);
		expect(uploadFileStem('Peli (2020).mkv')).toBe('Peli (2020)');
		expect(uploadFileStem('Peli.es.srt')).toBe('Peli.es');
	});

	it('empareja subtítulos con el vídeo de la cola', () => {
		expect(subtitleMatchesVideo('Peli.es.srt', 'Peli.mkv')).toBe(true);
		expect(subtitleMatchesVideo('Peli.srt', 'Peli.mkv')).toBe(true);
		expect(subtitleMatchesVideo('Peli.srt', 'Otra.mkv')).toBe(false);
		expect(subtitleMatchesVideo('Peli.2020.es.srt', 'Peli.2020.mkv')).toBe(true);
		expect(subtitleMatchesVideo('Peli.2020.es.srt', 'Peli.mkv')).toBe(false);
		expect(subtitleMatchesVideo('Otra.srt', 'Peli.mkv')).toBe(false);
	});

	it('asigna carpeta a cada vídeo y a sus subtítulos', () => {
		const videos = [
			{ name: 'Peli.mkv', folder: 'Peli' },
			{ name: 'Otra.mp4', folder: 'Otra carpeta' }
		];
		expect(uploadSubdirForFile('Peli.mkv', true, videos)).toBe('Peli');
		expect(uploadSubdirForFile('Peli.es.srt', true, videos)).toBe('Peli');
		expect(uploadSubdirForFile('Otra.srt', true, videos)).toBe('Otra carpeta');
		expect(
			uploadSubdirForFile('Peli.2020.es.srt', true, [
				{ name: 'Peli.mkv', folder: 'Peli' },
				{ name: 'Peli.2020.mkv', folder: 'Peli 2020' }
			])
		).toBe('Peli 2020');
		expect(uploadSubdirForFile('Suelta.srt', true, videos)).toBe('');
		expect(uploadSubdirForFile('Peli.mkv', false, videos)).toBe('');
	});

	it('valida nombres como la subida genérica', () => {
		expect(validUploadComponent('Peli (2020)')).toBe(true);
		expect(validUploadComponent('')).toBe(false);
		expect(validUploadComponent('.oculta')).toBe(false);
		expect(validUploadComponent('sub/peli')).toBe(false);
		expect(validUploadComponent('sub\\peli')).toBe(false);
	});
});
