import { env } from '$env/dynamic/private';

// El proyecto no instala @types/node; estas APIs solo se ejecutan en el
// servidor de SvelteKit (adapter-node).
// @ts-expect-error falta la declaración de node:http
import http from 'node:http';
// @ts-expect-error falta la declaración de node:https
import https from 'node:https';
// @ts-expect-error falta la declaración de node:stream
import { Readable } from 'node:stream';

const DEFAULT_ERROR = 'No se pudo contactar con el servidor para subir el archivo.';

export interface UploadProxyOptions {
	token: string;
	contentLength: string;
	errorMessage: string;
	origin?: string;
	publicBase?: string;
}

function errorResponse(message: string): Response {
	return new Response(JSON.stringify({ error: message }), {
		status: 502,
		headers: {
			'Content-Type': 'application/json; charset=utf-8',
			'Cache-Control': 'no-store'
		}
	});
}

function normalizedBase(value: string | undefined): string {
	return String(value ?? '')
		.trim()
		.replace(/\/+$/, '');
}

// resolveUploadTarget decide a qué servidor se reenvía la subida. La variable
// privada INTERNAL_API_URL evita que el tráfico entre y salga otra vez por
// Apache; sin ella se conserva la URL pública configurada.
export function resolveUploadTarget(target: string, origin?: string, publicBase?: string): string {
	if (/^https?:\/\//i.test(target)) {
		return target;
	}
	const internal = normalizedBase(env.INTERNAL_API_URL);
	const base = internal || normalizedBase(publicBase);
	const path = target.startsWith('/') ? target : `/${target}`;
	if (base) {
		return `${base}${path}`;
	}
	return new URL(path, origin ?? 'http://localhost').toString();
}

// proxyUpload reenvía un PUT en streaming usando node:http/node:https. A
// diferencia de fetch, no tiene límite de espera de cabeceras y no carga el
// archivo en memoria, así que una subida larga no se corta a los 300 s.
export async function proxyUpload(
	request: Request,
	path: string,
	proxyOptions: UploadProxyOptions
): Promise<Response> {
	const options = { ...proxyOptions, errorMessage: proxyOptions.errorMessage || DEFAULT_ERROR };

	let target: URL;
	try {
		target = new URL(resolveUploadTarget(path, options.origin, options.publicBase));
	} catch {
		return errorResponse(options.errorMessage);
	}
	if (target.protocol !== 'http:' && target.protocol !== 'https:') {
		return errorResponse(options.errorMessage);
	}

	let source: any;
	try {
		source = request.body ? Readable.fromWeb(request.body as any) : Readable.from([]);
	} catch {
		return errorResponse(options.errorMessage);
	}

	const transport: any = target.protocol === 'https:' ? https : http;

	const signal = request.signal;
	return new Promise<Response>((resolve) => {
		let settled = false;
		let clientRequest: any = null;

		const cleanup = () => {
			signal?.removeEventListener('abort', onAbort);
			source.removeListener?.('error', onSourceError);
		};
		const fail = () => {
			if (settled) {
				return;
			}
			settled = true;
			cleanup();
			source.destroy?.();
			clientRequest?.destroy();
			resolve(errorResponse(options.errorMessage));
		};
		const onAbort = () => {
			// Destruir la conexión con el servidor hace que Go elimine su .part.
			source.destroy?.();
			clientRequest?.destroy();
			fail();
		};
		const onSourceError = () => {
			fail();
		};

		source.on?.('error', onSourceError);
		signal?.addEventListener('abort', onAbort, { once: true });

		try {
			clientRequest = transport.request(
				target.toString(),
				{
					method: 'PUT',
					agent: false,
					headers: {
						Authorization: `Bearer ${options.token}`,
						'Content-Length': options.contentLength,
						'Content-Type': 'application/octet-stream',
						Accept: 'application/json'
					}
				},
				(serverResponse: any) => {
					if (settled) {
						return;
					}
					settled = true;
					cleanup();
					source.unpipe?.(clientRequest);
					source.destroy?.();

					const status = serverResponse.statusCode ?? 502;
					const contentType = Array.isArray(serverResponse.headers['content-type'])
						? serverResponse.headers['content-type'][0]
						: serverResponse.headers['content-type'] || 'application/json; charset=utf-8';
					const headers = {
						'Content-Type': contentType,
						'Cache-Control': 'no-store'
					};
					if (status === 204 || status === 304) {
						resolve(new Response(null, { status, headers }));
						return;
					}
					resolve(
						new Response(Readable.toWeb(serverResponse) as ReadableStream<Uint8Array>, {
							status,
							headers
						})
					);
				}
			);
		} catch {
			fail();
			return;
		}
		if (settled) {
			return;
		}

		clientRequest.setTimeout(0);
		clientRequest.on('socket', (socket: any) => {
			socket.setTimeout(0);
		});
		clientRequest.on('error', fail);

		if (signal?.aborted) {
			onAbort();
			return;
		}

		source.pipe(clientRequest);
	});
}
