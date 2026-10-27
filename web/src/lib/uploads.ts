import { formatBytes } from './media';

export const UPLOAD_VIDEO_EXTENSIONS = [
	'.mkv',
	'.mp4',
	'.avi',
	'.mov',
	'.m4v',
	'.ts',
	'.m2ts'
] as const;

export const UPLOAD_SUBTITLE_EXTENSIONS = [
	'.srt',
	'.vtt',
	'.ass',
	'.ssa',
	'.sub',
	'.idx'
] as const;

export const UPLOAD_ACCEPT = [...UPLOAD_VIDEO_EXTENSIONS, ...UPLOAD_SUBTITLE_EXTENSIONS].join(',');

export function isUploadableFileName(name: string): boolean {
	const lower = name.toLowerCase();
	return [...UPLOAD_VIDEO_EXTENSIONS, ...UPLOAD_SUBTITLE_EXTENSIONS].some((extension) =>
		lower.endsWith(extension)
	);
}

export function formatUploadSize(bytes: number): string {
	return formatBytes(bytes);
}

export function uploadExtension(name: string): string {
	const dot = name.lastIndexOf('.');
	return dot >= 0 ? name.slice(dot).toLowerCase() : '';
}

function subtitleLanguageSuffix(name: string, extension: string): string {
	if (!(UPLOAD_SUBTITLE_EXTENSIONS as readonly string[]).includes(extension)) {
		return '';
	}
	const stem = name.slice(0, name.length - extension.length);
	const dot = stem.lastIndexOf('.');
	if (dot < 0) {
		return '';
	}
	const language = stem.slice(dot + 1);
	return /^[A-Za-z]{2,3}$/.test(language) ? `.${language.toLowerCase()}` : '';
}

export function buildEpisodeUploadName(
	seriesName: string,
	season: number,
	episode: number,
	originalName: string
): string {
	const extension = uploadExtension(originalName);
	const suffix = subtitleLanguageSuffix(originalName, extension);
	return `${seriesName} S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}${suffix}${extension}`;
}

export interface UploadURLParameters {
	library: number;
	dir: string;
	name: string;
	subdir?: string;
}

export function buildUploadURL({ library, dir, name, subdir }: UploadURLParameters): string {
	const params = new URLSearchParams({
		library: String(library),
		dir,
		name
	});
	const cleanSubdir = subdir?.trim() ?? '';
	if (cleanSubdir) {
		params.set('subdir', cleanSubdir);
	}
	return `/admin/libraries/upload?${params.toString()}`;
}

function serverErrorMessage(payload: unknown): string {
	if (typeof payload === 'object' && payload !== null) {
		const error = (payload as { error?: unknown }).error;
		if (typeof error === 'string' && error.trim() !== '') {
			return error;
		}
	}
	return '';
}

export function uploadErrorMessage(status: number, payload: unknown): string {
	if (status === 403) {
		return 'No se pudo subir el archivo. Activa «Permitir cambiar archivos» en Administración → Biblioteca.';
	}
	const serverMessage = serverErrorMessage(payload);
	if (serverMessage) {
		return serverMessage;
	}
	if (status === 411) {
		return 'El navegador no indicó el tamaño del archivo.';
	}
	if (status === 413) {
		return 'El archivo supera el tamaño máximo configurado.';
	}
	if (status === 507) {
		return 'No hay espacio libre suficiente en el disco de destino.';
	}
	if (status === 409) {
		return 'El archivo ya existe o hay otra subida en curso.';
	}
	return 'No se pudo subir el archivo.';
}
