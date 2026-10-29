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
