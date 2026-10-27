import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { env } from '$env/dynamic/private';
// @ts-expect-error falta la declaración de node:http
import http from 'node:http';
import { proxyUpload, resolveUploadTarget } from './upload-proxy';

const servers: any[] = [];

async function startServer(handler: (request: any, response: any) => void): Promise<string> {
	const server = http.createServer(handler);
	servers.push(server);
	await new Promise<void>((resolve, reject) => {
		server.once('error', reject);
		server.listen(0, '127.0.0.1', () => resolve());
	});
	const address = server.address() as { port: number };
	return `http://127.0.0.1:${address.port}`;
}

async function closeServers(): Promise<void> {
	await Promise.all(
		servers.splice(0).map(
			(server) =>
				new Promise<void>((resolve) => {
					server.close(() => resolve());
				})
		)
	);
}

beforeEach(() => {
	delete env.INTERNAL_API_URL;
});

afterEach(async () => {
	delete env.INTERNAL_API_URL;
	await closeServers();
});

describe('proxy de subidas', () => {
	it('reenvía el PUT en streaming y devuelve estado y cuerpo', async () => {
		let received: { headers?: any; body?: string } = {};
		const target = await startServer((request, response) => {
			let body = '';
			request.setEncoding('utf8');
			request.on('data', (chunk: string) => {
				body += chunk;
			});
			request.on('end', () => {
				received = { headers: request.headers, body };
				response.writeHead(201, { 'Content-Type': 'application/json' });
				response.end('{"ok":true}');
			});
		});
		env.INTERNAL_API_URL = target;

		const request = new Request('http://localhost/subir', {
			method: 'PUT',
			body: 'abcdef',
			headers: { 'Content-Length': '6' }
		});
		const response = await proxyUpload(request, '/api/v1/prueba?x=1', {
			token: 'token-123',
			contentLength: '6',
			errorMessage: 'No se pudo contactar.'
		});

		expect(response.status).toBe(201);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(response.headers.get('Content-Type')).toBe('application/json');
		expect(await response.text()).toBe('{"ok":true}');
		expect(received.body).toBe('abcdef');
		expect(received.headers.authorization).toBe('Bearer token-123');
		expect(received.headers['content-length']).toBe('6');
		expect(received.headers['content-type']).toBe('application/octet-stream');
		expect(received.headers.accept).toBe('application/json');
	});

	it('usa la URL pública o el origen si no hay INTERNAL_API_URL', () => {
		expect(resolveUploadTarget('/api/v1/x', 'http://localhost:5173', '')).toBe(
			'http://localhost:5173/api/v1/x'
		);
		expect(resolveUploadTarget('/api/v1/x', 'http://localhost:5173', 'https://api.test')).toBe(
			'https://api.test/api/v1/x'
		);
		env.INTERNAL_API_URL = 'http://servidor:8080/';
		expect(resolveUploadTarget('/api/v1/x', 'http://localhost:5173', 'https://api.test')).toBe(
			'http://servidor:8080/api/v1/x'
		);
	});

	it('devuelve 502 con el mensaje del endpoint si no puede conectar', async () => {
		const target = await startServer(() => {});
		await closeServers();
		env.INTERNAL_API_URL = target;

		const response = await proxyUpload(
			new Request('http://localhost/subir', { method: 'PUT', body: 'x' }),
			'/api/v1/prueba',
			{
				token: 'token-123',
				contentLength: '1',
				errorMessage: 'No se pudo contactar con el servidor.'
			}
		);
		expect(response.status).toBe(502);
		expect(await response.json()).toEqual({ error: 'No se pudo contactar con el servidor.' });
	});

	it('destruye la petición al servidor si el navegador cancela', async () => {
		let requestStarted: () => void = () => {};
		const started = new Promise<void>((resolve) => {
			requestStarted = resolve;
		});
		let requestClosed: () => void = () => {};
		const closed = new Promise<void>((resolve) => {
			requestClosed = resolve;
		});
		const target = await startServer((request) => {
			requestStarted();
			request.on('data', () => {});
			request.on('aborted', requestClosed);
			request.on('close', requestClosed);
		});
		env.INTERNAL_API_URL = target;

		const controller = new AbortController();
		const body = new ReadableStream<Uint8Array>({
			start(stream) {
				stream.enqueue(new TextEncoder().encode('x'));
			}
		});
		const request = new Request('http://localhost/subir', {
			method: 'PUT',
			body,
			duplex: 'half',
			signal: controller.signal
		} as RequestInit & { duplex: 'half' });
		const pending = proxyUpload(request, '/api/v1/prueba', {
			token: 'token-123',
			contentLength: '100',
			errorMessage: 'No se pudo contactar.'
		});

		await started;
		controller.abort();
		await pending;
		await Promise.race([
			closed,
			new Promise<void>((_, reject) => {
				setTimeout(() => reject(new Error('el servidor no vio la cancelación')), 2000);
			})
		]);
	});
});
