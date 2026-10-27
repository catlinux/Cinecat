import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { Person, PersonMedia } from '$lib/api';
import PeoplePage from './+page.svelte';
import { load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const person: Person = {
	id: 3084,
	name: 'Marlon Brando',
	biography:
		'Actor estadounidense considerado uno de los mejores intérpretes de la historia del cine. ' +
		'Ganó dos premios Óscar y participó en películas fundamentales de la segunda mitad del siglo XX. ' +
		'Su estilo interpretativo cambió la forma de entender el oficio y dejó papeles inolvidables. ' +
		'Esta biografía se alarga para que la ficha tenga que mostrar el desplegable de «Leer más».',
	birthday: '1924-04-03',
	deathday: '2004-07-01',
	place_of_birth: 'Omaha, Nebraska, Estados Unidos',
	photo_url: 'https://images.example/marlon.jpg',
	known_for: 'El padrino, Apocalypse Now'
};

const inLibrary: PersonMedia[] = [
	{
		type: 'movie',
		id: 7,
		tmdb_id: 238,
		title: 'El padrino',
		year: 1972,
		poster_url: 'https://images.example/padrino.jpg',
		rating: 9.2,
		roles: ['Vito Corleone']
	},
	{
		type: 'series',
		id: 5,
		title: 'Una serie',
		year: 2001,
		poster_url: null,
		rating: null,
		roles: ['Director']
	}
];

const notInLibrary: PersonMedia[] = [
	{
		type: 'movie',
		tmdb_id: 999,
		title: 'Película que no tienes',
		year: 1980,
		poster_url: 'https://images.example/otra.jpg',
		rating: 7.1,
		roles: ['Director']
	}
];

function loadEvent(fetchImpl: typeof fetch, token: string | null = 'token-123', id = '3084') {
	return {
		params: { id },
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(`http://localhost/people/${id}`),
		cookies: {},
		route: { id: '/people/[id]' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function pageData(overrides: Record<string, unknown> = {}) {
	return {
		person,
		inLibrary,
		notInLibrary,
		errorMessage: null,
		...overrides
	};
}

describe('carga de /people/[id]', () => {
	it('pide la persona con la sesión y devuelve las dos secciones', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ person, in_library: inLibrary, not_in_library: notInLibrary })
		);

		const result = (await load(loadEvent(fetchMock) as never)) as {
			person: Person;
			inLibrary: PersonMedia[];
			notInLibrary: PersonMedia[];
			errorMessage: null;
		};

		expect(result.person.name).toBe('Marlon Brando');
		expect(result.inLibrary).toEqual(inLibrary);
		expect(result.notInLibrary).toEqual(notInLibrary);
		expect(result.errorMessage).toBeNull();
		const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('http://api.test/api/v1/people/3084');
		expect(new Headers(init.headers).get('Authorization')).toBe('Bearer token-123');
	});

	it('devuelve 404 si la persona no existe', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({ status: 404 });
	});

	it('devuelve un estado de error controlado si TMDB no responde', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin TMDB' }, 502));
		const result = (await load(loadEvent(fetchMock) as never)) as {
			person: null;
			inLibrary: PersonMedia[];
			notInLibrary: PersonMedia[];
			errorMessage: string;
		};
		expect(result.person).toBeNull();
		expect(result.inLibrary).toEqual([]);
		expect(result.notInLibrary).toEqual([]);
		expect(result.errorMessage).toContain('TMDB no responde');
	});

	it('redirige al login sin sesión', async () => {
		await expect(
			load(loadEvent(vi.fn() as never, null) as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('página de persona', () => {
	it('muestra cabecera, biografía y las dos secciones separadas', () => {
		const { body } = render(PeoplePage, {
			props: { data: pageData() as never }
		});

		expect(body).toContain('Marlon Brando');
		expect(body).toContain('Omaha, Nebraska, Estados Unidos');
		expect(body).toContain('Conocido por:');
		expect(body).toContain('Leer más');
		expect(body).toContain('En tu biblioteca');
		expect(body).toContain('No la tienes');
		expect(body).toContain('href="/items/7"');
		expect(body).toContain('href="/series/5"');
		expect(body).toContain('como Vito Corleone');
		expect(body).toContain('Dirección');
		expect(body).toContain('Película que no tienes');
		expect(body).toContain('Estos títulos no están en tu biblioteca');
		expect(body.indexOf('En tu biblioteca')).toBeLessThan(body.indexOf('No la tienes'));
	});

	it('muestra mensajes breves si alguna sección está vacía', () => {
		const { body } = render(PeoplePage, {
			props: {
				data: pageData({
					inLibrary: [],
					notInLibrary: []
				}) as never
			}
		});
		expect(body).toContain('No aparece en ninguna película o serie de tu biblioteca');
		expect(body).toContain('No hay otros títulos suyos para mostrar');
	});

	it('muestra el aviso de TMDB sin romper la página', () => {
		const { body } = render(PeoplePage, {
			props: {
				data: {
					person: null,
					inLibrary: [],
					notInLibrary: [],
					errorMessage: 'TMDB no responde ahora; inténtalo más tarde.'
				} as never
			}
		});
		expect(body).toContain('TMDB no responde ahora');
		expect(body).toContain('role="alert"');
	});
});
