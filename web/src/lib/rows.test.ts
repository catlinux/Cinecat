import { describe, expect, it } from 'vitest';
import { creditRows, crewPriority, jobLabel, personInitials, rowScrollState } from './rows';

describe('estado de desplazamiento de una fila', () => {
	it('calcula los extremos y una página de avance', () => {
		expect(rowScrollState(0, 600, 1800)).toMatchObject({
			canPrev: false,
			canNext: true,
			prevOffset: 0,
			nextOffset: 600,
			overflow: true
		});
		const middle = rowScrollState(600, 600, 1800);
		expect(middle.canPrev).toBe(true);
		expect(middle.canNext).toBe(true);
		expect(middle.prevOffset).toBe(0);
		expect(middle.nextOffset).toBe(1200);
		const end = rowScrollState(1200, 600, 1800);
		expect(end).toMatchObject({
			canPrev: true,
			canNext: false,
			prevOffset: 600,
			nextOffset: 1200
		});
	});

	it('no muestra flechas si el contenido cabe', () => {
		expect(rowScrollState(0, 900, 899)).toMatchObject({
			canPrev: false,
			canNext: false,
			overflow: false
		});
	});

	it('tolera redondeos de un píxel', () => {
		const state = rowScrollState(1, 900, 1800);
		expect(state.canPrev).toBe(false);
		const end = rowScrollState(899, 900, 1800);
		expect(end.canNext).toBe(false);
	});
});

describe('orden del reparto y del equipo', () => {
	it('pone primero el reparto y ordena el equipo por importancia', () => {
		const rows = creditRows(
			{
				cast: [
					{
						id: 101,
						name: 'Protagonista',
						character: 'Heroína',
						photo_url: 'https://images.example/p.jpg'
					}
				],
				crew: [
					{ id: 102, name: 'Productora', job: 'Producer', photo_url: '' },
					{ id: 103, name: 'Directora', job: 'Director', photo_url: '' },
					{ name: 'Guionista', job: 'Writer', photo_url: '' },
					{ name: 'Montadora', job: 'Editor', photo_url: '' },
					{ name: 'Creadora', job: 'Creator', photo_url: '' },
					{ name: 'Extra', job: 'Gaffer', photo_url: '' }
				]
			},
			[],
			[]
		);
		expect(rows.map((row) => row.name)).toEqual([
			'Protagonista',
			'Directora',
			'Guionista',
			'Productora',
			'Montadora',
			'Creadora',
			'Extra'
		]);
		expect(rows[0].subtitle).toBe('como Heroína');
		expect(rows[0].id).toBe(101);
		expect(rows[1].subtitle).toBe('Dirección');
		expect(rows[1].id).toBe(103);
	});

	it('con nombres de respaldo pone el reparto antes que la dirección', () => {
		const rows = creditRows(null, ['Actor', 'Actriz'], ['Director']);
		expect(rows.map((row) => row.name)).toEqual(['Actor', 'Actriz', 'Director']);
		expect(rows[2].subtitle).toBe('Dirección');
		expect(rows[0].id).toBeNull();
		expect(rows[2].id).toBeNull();
	});

	it('asigna prioridad a los oficios conocidos', () => {
		expect(crewPriority('Director')).toBeLessThan(crewPriority('Writer'));
		expect(crewPriority('Writer')).toBeLessThan(crewPriority('Producer'));
		expect(crewPriority('Editor')).toBeLessThan(crewPriority('Creator'));
		expect(crewPriority('Creator')).toBeLessThan(crewPriority('Gaffer'));
	});
});

describe('etiquetas y retratos', () => {
	it('traduce los trabajos conocidos', () => {
		expect(jobLabel('Director')).toBe('Dirección');
		expect(jobLabel('Writer')).toBe('Guion');
		expect(jobLabel('Executive Producer')).toBe('Producción');
		expect(jobLabel('Original Music Composer')).toBe('Música');
		expect(jobLabel('Director of Photography')).toBe('Fotografía');
		expect(jobLabel('Editor')).toBe('Montaje');
		expect(jobLabel('Creator')).toBe('Creación');
	});

	it('calcula iniciales con una o varias palabras', () => {
		expect(personInitials('Ana')).toBe('A');
		expect(personInitials('Ana María López')).toBe('AL');
		expect(personInitials('Ángel Ibáñez')).toBe('AI');
		expect(personInitials('   ')).toBe('');
	});
});
