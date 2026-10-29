package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.CollectionDetail
import io.github.catlinux.cinecat.core.api.Credits
import io.github.catlinux.cinecat.core.api.MediaItem

internal fun sampleMediaItem(
    id: Long,
    favorite: Boolean = false,
    watchlist: Boolean = false,
    watched: Boolean = false
): MediaItem = MediaItem(
    id = id,
    libraryId = 1,
    libraryKind = "video",
    title = "Pelicula $id",
    fileTitle = "Pelicula $id",
    year = 2024,
    kind = "movie",
    season = null,
    episode = null,
    tmdbId = null,
    overview = "Sinopsis de prueba",
    posterUrl = null,
    originalTitle = null,
    tagline = null,
    genres = emptyList(),
    runtimeMinutes = 112,
    rating = 7.5,
    directors = emptyList(),
    cast = emptyList(),
    credits = Credits(emptyList(), emptyList()),
    backdropUrl = null,
    watched = watched,
    favorite = favorite,
    watchlist = watchlist,
    progress = null,
    collections = emptyList(),
    seriesId = null,
    seriesTitle = null,
    episodeTitle = null,
    stillUrl = null,
    seriesPosterUrl = null,
    seriesBackdropUrl = null
)

internal fun sampleCollectionDetail(
    id: Long = 9,
    favorite: Boolean = false,
    watchlist: Boolean = false
): CollectionDetail = CollectionDetail(
    id = id,
    name = "Coleccion",
    overview = null,
    posterUrl = null,
    backdropUrl = null,
    itemCount = 0,
    source = "manual",
    edited = false,
    favorite = favorite,
    watchlist = watchlist,
    items = emptyList()
)

internal fun sampleSeason(
    season: Int,
    episodeCount: Int = 2,
    watchedCount: Int = 0
): io.github.catlinux.cinecat.core.api.Season =
    io.github.catlinux.cinecat.core.api.Season(
        season = season,
        name = if (season == 0) "Especiales" else "Temporada $season",
        overview = null,
        posterUrl = null,
        episodeCount = episodeCount.toLong(),
        watchedCount = watchedCount.toLong()
    )

internal fun sampleEpisode(
    id: Long,
    season: Int,
    episode: Int,
    watched: Boolean = false
): MediaItem = sampleMediaItem(id, watched = watched).copy(
    kind = "episode",
    season = season,
    episode = episode,
    episodeTitle = "Episodio $episode",
    seriesId = 3
)

internal fun sampleSeasonDetail(
    season: Int,
    items: List<MediaItem>
): io.github.catlinux.cinecat.core.api.SeasonDetail =
    io.github.catlinux.cinecat.core.api.SeasonDetail(
        season = season,
        name = if (season == 0) "Especiales" else "Temporada $season",
        overview = null,
        posterUrl = null,
        items = items
    )

internal fun sampleSeriesDetail(
    seasons: List<io.github.catlinux.cinecat.core.api.Season>,
    nextEpisode: MediaItem? = null
): io.github.catlinux.cinecat.core.api.SeriesDetail =
    io.github.catlinux.cinecat.core.api.SeriesDetail(
        id = 3,
        title = "Serie",
        fileTitle = "Serie",
        originalTitle = null,
        year = 2020,
        tagline = null,
        overview = "Sinopsis de la serie",
        posterUrl = null,
        backdropUrl = null,
        genres = emptyList(),
        rating = 7.0,
        directors = emptyList(),
        cast = emptyList(),
        credits = Credits(emptyList(), emptyList()),
        tmdbId = null,
        seasonCount = seasons.size.toLong(),
        episodeCount = seasons.sumOf { it.episodeCount },
        watchedCount = seasons.sumOf { it.watchedCount },
        favorite = false,
        watchlist = false,
        seasons = seasons,
        nextEpisode = nextEpisode
    )
