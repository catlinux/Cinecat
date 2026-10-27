import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import PersonCard from './PersonCard.svelte';

describe('PersonCard', () => {
	it('enlaza toda la tarjeta cuando tiene id de persona', () => {
		const { body } = render(PersonCard, {
			props: {
				name: 'Marlon Brando',
				subtitle: 'como Vito Corleone',
				href: '/people/3084'
			}
		});

		expect(body).toContain('href="/people/3084"');
		expect(body).toContain('person-card-link');
		expect(body).toContain('Marlon Brando');
	});

	it('mantiene una tarjeta sin enlace cuando no hay id', () => {
		const { body } = render(PersonCard, {
			props: { name: 'Marlon Brando' }
		});

		expect(body).toContain('<article');
		expect(body).not.toContain('person-card-link');
		expect(body).not.toContain('/people/');
	});
});
