import { describe, expect, it } from 'vitest';
import {
	UPLOAD_ACCEPT,
	buildUploadURL,
	formatUploadSize,
	isUploadableFileName,
	uploadErrorMessage
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
