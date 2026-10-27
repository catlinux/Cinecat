import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { UploadTargetRoot } from '$lib/api';
import EpisodeUploadDialog from './EpisodeUploadDialog.svelte';

const roots: UploadTargetRoot[] = [
	{
		library_id: 1,
		library_name: 'Series',
		library_kind: 'video',
		path: '/media/series-a',
		movie_count: 0,
		series_count: 2
	},
	{
		library_id: 1,
		library_name: 'Series',
		library_kind: 'video',
		path: '/media/series-b',
		movie_count: 0,
		series_count: 7
	}
];

describe('EpisodeUploadDialog', () => {
	it('en serie nueva pide nombre y destino, con la raíz de más series por defecto', () => {
		const { body } = render(EpisodeUploadDialog, {
			props: { mode: 'serie-nueva', roots }
		});

		expect(body).toContain('Crear nueva serie');
		expect(body).toContain('Nombre de la serie');
		expect(body).toContain('Serie de destino');
		expect(body).toContain('/media/series-a');
		expect(body).toContain('/media/series-b');
		expect(body).toContain('value="/media/series-b" selected');
		expect(body).toContain('No cierres la página hasta que terminen las subidas.');
	});

	it('en temporada nueva pide el número y en capítulo mantiene la temporada fija', () => {
		const season = render(EpisodeUploadDialog, {
			props: { mode: 'temporada-nueva', seriesId: 5, defaultSeason: 3 }
		}).body;
		expect(season).toContain('Número de temporada');
		expect(season).toContain('value="3"');
		expect(season).toContain('Nueva temporada');

		const chapter = render(EpisodeUploadDialog, {
			props: { mode: 'capitulo', seriesId: 5, season: 2 }
		}).body;
		expect(chapter).toContain('Subir capítulo');
		expect(chapter).not.toContain('Número de temporada');
		expect(chapter).not.toContain('Nombre de la serie');
	});
});
