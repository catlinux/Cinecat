import type { ApiClient, MediaItem, SeriesDetail } from '$lib/api';
import { episodeNeighbors } from '$lib/media';
import type { EpisodeNeighbors } from '$lib/media';

export async function loadEpisodeNavigation(
	api: ApiClient,
	item: MediaItem,
	token: string
): Promise<EpisodeNeighbors | null> {
	if (item.kind !== 'episode' || item.series_id === null || item.season === null) {
		return null;
	}

	const seriesID = item.series_id;
	let series: SeriesDetail;
	try {
		series = await api.getSeriesDetail(seriesID, token);
	} catch {
		return null;
	}

	const orderedSeasons = [...(series.seasons ?? [])].sort(
		(left, right) => left.season - right.season
	);
	const seasonIndex = orderedSeasons.findIndex((season) => season.season === item.season);
	if (seasonIndex === -1) {
		return null;
	}

	let currentItems: MediaItem[];
	try {
		const currentSeason = await api.getSeriesSeason(seriesID, item.season, token);
		currentItems = currentSeason.items ?? [];
	} catch {
		return null;
	}

	if (!currentItems.some((episode) => episode.id === item.id)) {
		return null;
	}

	const itemsBySeason = new Map<number, MediaItem[]>([[item.season, currentItems]]);
	let neighbors = episodeNeighbors(item, orderedSeasons, itemsBySeason);

	if (neighbors.previous === null && seasonIndex > 0) {
		const previousSeason = orderedSeasons[seasonIndex - 1].season;
		try {
			const result = await api.getSeriesSeason(seriesID, previousSeason, token);
			itemsBySeason.set(previousSeason, result.items ?? []);
			neighbors = episodeNeighbors(item, orderedSeasons, itemsBySeason);
		} catch {
			// La navegación es un extra: si la temporada contigua falla, se omite.
		}
	}

	if (neighbors.next === null && seasonIndex < orderedSeasons.length - 1) {
		const nextSeason = orderedSeasons[seasonIndex + 1].season;
		try {
			const result = await api.getSeriesSeason(seriesID, nextSeason, token);
			itemsBySeason.set(nextSeason, result.items ?? []);
			neighbors = episodeNeighbors(item, orderedSeasons, itemsBySeason);
		} catch {
			// Igual que con la temporada anterior, la ficha no depende de esto.
		}
	}

	return neighbors;
}
