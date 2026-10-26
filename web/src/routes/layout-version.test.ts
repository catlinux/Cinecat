import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import { versionMismatch, webVersion } from '$lib/version';
import Layout from './+layout.svelte';
import { load } from './+layout.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const children = createRawSnippet(() => ({
	render: () => '<p>Contenido</p>'
}));

function layoutData(serverVersion: string | null, serverCommit: string | null = null) {
	return {
		hasSession: false,
		isAdmin: false,
		serverVersion,
		serverCommit,
		webVersion
	};
}

function renderFooter(serverVersion: string | null, serverCommit: string | null = null): string {
	return render(Layout, {
		props: {
			data: layoutData(serverVersion, serverCommit),
			children
		}
	}).body;
}

function visibleText(html: string): string {
	return html.replace(/<!--[\s\S]*?-->/g, '');
}

describe('cálculo del aviso de versión', () => {
	it('no avisa si las versiones coinciden', () => {
		expect(versionMismatch(webVersion, webVersion)).toBe(false);
	});

	it('avisa si las versiones son distintas', () => {
		expect(versionMismatch('0.11.0', webVersion)).toBe(true);
	});

	it('no avisa si el servidor no responde', () => {
		expect(versionMismatch(null, webVersion)).toBe(false);
	});
});

describe('carga de la versión del servidor', () => {
	it('deja la versión nula sin romper la carga si la API falla', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock
		} as never)) as { serverVersion: string | null; webVersion: string };

		expect(result.serverVersion).toBeNull();
		expect(result.webVersion).toBe(webVersion);
	});

	it('devuelve la versión y el commit del servidor', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({
				server: '0.11.0',
				commit: 'abc1234',
				built_at: '2026-10-26T09:00:00Z'
			})
		);

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock
		} as never)) as {
			serverVersion: string | null;
			serverCommit: string | null;
			webVersion: string;
		};

		expect(result.serverVersion).toBe('0.11.0');
		expect(result.serverCommit).toBe('abc1234');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/system/version');
	});
});

describe('pie de página', () => {
	it('muestra la versión del servidor y el commit', () => {
		const body = renderFooter(webVersion, 'abc1234');

		expect(visibleText(body)).toContain(`Cinecat ${webVersion} (abc1234) · GPL-3.0`);
		expect(body).toContain('Servidor:');
		expect(body).not.toContain('La web y el servidor tienen versiones distintas');
	});

	it('avisa de forma visible si las versiones no coinciden', () => {
		const body = renderFooter('0.11.0', 'def5678');

		expect(visibleText(body)).toContain('Cinecat 0.11.0 (def5678) · GPL-3.0');
		expect(body).toContain(
			'La web y el servidor tienen versiones distintas: recarga o actualiza.'
		);
		expect(body).toContain('role="status"');
	});

	it('usa la versión web y no avisa si el servidor no responde', () => {
		const body = renderFooter(null);

		expect(visibleText(body)).toContain(`Cinecat ${webVersion} · GPL-3.0`);
		expect(body).not.toContain('La web y el servidor tienen versiones distintas');
	});
});
