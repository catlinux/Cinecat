package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import io.github.catlinux.cinecat.core.api.CastCredit
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.Series
import io.github.catlinux.cinecat.core.api.SeriesDetail
import io.github.catlinux.cinecat.core.api.tmdbImage

@Composable
fun TvMovieDetailContent(
    state: MovieDetailUiState.Ready,
    horizontalPadding: Dp,
    backFocusRequester: FocusRequester,
    onPlay: (Long, Double) -> Unit,
    onOpenMovie: (Long) -> Unit,
    onOpenCollection: (Long) -> Unit,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit,
    onToggleWatched: () -> Unit
) {
    val item = state.item
    val listState = rememberLazyListState()
    val initialFocusRequester = remember(item.id) { FocusRequester() }
    val collectionFocusRequester = remember(item.id) { FocusRequester() }
    val similarFocusRequester = remember(item.id) { FocusRequester() }
    val collection = item.collections.firstOrNull()
    val position = item.progress?.positionSeconds?.takeIf { it > 0.0 }
    val nextFocusRequester = when {
        collection != null -> collectionFocusRequester
        state.similar.isNotEmpty() -> similarFocusRequester
        else -> null
    }

    LaunchedEffect(item.id) {
        runCatching { listState.scrollToItem(0) }
        withFrameNanos { }
        runCatching { initialFocusRequester.requestFocus() }
    }

    LazyColumn(
        state = listState,
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(bottom = 24.dp)
    ) {
        item(key = "tv-header") {
            TvDetailHeader(
                title = item.title,
                backdropUrl = tmdbImage(item.backdropUrl ?: item.posterUrl, "w1280"),
                posterUrl = tmdbImage(item.posterUrl, "w342"),
                meta = movieMetaLine(item),
                overview = item.overview,
                primaryLabel = if (position != null) {
                    stringResource(R.string.resume_from, formatPlaybackPosition(position))
                } else {
                    stringResource(R.string.play)
                },
                initialFocusRequester = initialFocusRequester,
                nextFocusRequester = nextFocusRequester,
                backFocusRequester = backFocusRequester,
                favorite = item.favorite,
                watchlist = item.watchlist,
                watched = item.watched,
                watchedLabel = stringResource(R.string.watched),
                onPrimary = { onPlay(item.id, position ?: 0.0) },
                onToggleFavorite = onToggleFavorite,
                onToggleWatchlist = onToggleWatchlist,
                onToggleWatched = onToggleWatched
            )
        }

        if (collection != null) {
            item(key = "tv-collection") {
                OutlinedButton(
                    onClick = { onOpenCollection(collection.id) },
                    modifier = Modifier
                        .padding(start = horizontalPadding, end = horizontalPadding, top = 16.dp)
                        .focusRequester(collectionFocusRequester)
                        .focusRing(RoundedCornerShape(50))
                ) {
                    Text(stringResource(R.string.part_of_collection, collection.name))
                }
            }
        }

        if (collection == null && state.similar.isNotEmpty()) {
            item(key = "tv-similar-title") {
                DetailSectionTitle(
                    title = stringResource(R.string.similar_section),
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp,
                        bottom = 8.dp
                    )
                )
            }
            item(key = "tv-similar-row") {
                TvSimilarMoviesRow(
                    similar = state.similar,
                    horizontalPadding = horizontalPadding,
                    firstFocusRequester = similarFocusRequester,
                    onOpenMovie = onOpenMovie
                )
            }
        }

        val cast = if (item.credits.cast.isNotEmpty()) {
            item.credits.cast
        } else {
            item.cast.map { name -> CastCredit(name = name, character = "", photoUrl = "") }
        }
        if (cast.isNotEmpty()) {
            item(key = "cast-title") {
                DetailSectionTitle(
                    title = stringResource(R.string.cast_section),
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp,
                        bottom = 8.dp
                    )
                )
            }
            item(key = "cast-row") {
                DetailCastRow(cast = cast, horizontalPadding = horizontalPadding)
            }
        }

        if (item.directors.isNotEmpty()) {
            item(key = "directors") {
                Column(
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp
                    ),
                    verticalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    DetailSectionTitle(title = stringResource(R.string.directors_section))
                    Text(
                        text = item.directors.joinToString(
                            stringResource(R.string.list_separator)
                        ),
                        style = MaterialTheme.typography.bodyLarge
                    )
                }
            }
        }

        if (collection != null && state.similar.isNotEmpty()) {
            item(key = "similar-title") {
                DetailSectionTitle(
                    title = stringResource(R.string.similar_section),
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp,
                        bottom = 8.dp
                    )
                )
            }
            item(key = "similar-row") {
                TvSimilarMoviesRow(
                    similar = state.similar,
                    horizontalPadding = horizontalPadding,
                    firstFocusRequester = null,
                    onOpenMovie = onOpenMovie
                )
            }
        }
    }
}

@Composable
fun TvSeriesDetailContent(
    state: SeriesDetailUiState.Ready,
    horizontalPadding: Dp,
    backFocusRequester: FocusRequester,
    onSelectSeason: (Int) -> Unit,
    onRetrySeason: () -> Unit,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit,
    onToggleSeriesWatched: () -> Unit,
    onToggleSeasonWatched: () -> Unit,
    onToggleEpisodeWatched: (Long) -> Unit,
    onPlayEpisode: (Long, Double) -> Unit,
    onOpenSeries: (Long) -> Unit
) {
    val series = state.series
    val listState = rememberLazyListState()
    val initialFocusRequester = remember(series.id) { FocusRequester() }
    val firstTabFocusRequester = remember(series.id) { FocusRequester() }
    val similarFocusRequester = remember(series.id) { FocusRequester() }
    val nextEpisode = series.nextEpisode
    val hasSeasons = state.seasons.isNotEmpty()
    val nextFocusRequester = when {
        hasSeasons -> firstTabFocusRequester
        state.similar.isNotEmpty() -> similarFocusRequester
        else -> null
    }
    val nextCode = nextEpisode?.let { next ->
        formatEpisodeCode(next.season, next.episode).ifBlank {
            next.episodeTitle ?: next.title
        }
    }
    val primaryLabel = nextCode?.let { code ->
        stringResource(R.string.play_episode, code)
    }

    LaunchedEffect(series.id) {
        runCatching { listState.scrollToItem(0) }
        withFrameNanos { }
        runCatching { initialFocusRequester.requestFocus() }
    }

    LazyColumn(
        state = listState,
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(bottom = 24.dp)
    ) {
        item(key = "tv-header") {
            TvDetailHeader(
                title = series.title,
                backdropUrl = tmdbImage(series.backdropUrl ?: series.posterUrl, "w1280"),
                posterUrl = tmdbImage(series.posterUrl, "w342"),
                meta = seriesMetaLine(series),
                overview = series.overview,
                primaryLabel = primaryLabel,
                initialFocusRequester = initialFocusRequester,
                nextFocusRequester = nextFocusRequester,
                backFocusRequester = backFocusRequester,
                favorite = series.favorite,
                watchlist = series.watchlist,
                watched = state.seriesWatched,
                watchedLabel = stringResource(R.string.series_watched),
                onPrimary = nextEpisode?.let { next ->
                    { onPlayEpisode(next.id, next.progress?.positionSeconds ?: 0.0) }
                },
                onToggleFavorite = onToggleFavorite,
                onToggleWatchlist = onToggleWatchlist,
                onToggleWatched = onToggleSeriesWatched
            )
        }

        if (hasSeasons) {
            item(key = "season-tabs") {
                SeasonTabs(
                    seasons = state.seasons,
                    selectedSeason = state.selectedSeason,
                    horizontalPadding = horizontalPadding,
                    onSelectSeason = onSelectSeason,
                    firstTabFocusRequester = firstTabFocusRequester
                )
            }
            when (val episodes = state.episodes) {
                SeasonEpisodesState.None -> Unit
                SeasonEpisodesState.Loading -> item(key = "season-loading") {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(32.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        CircularProgressIndicator()
                    }
                }
                SeasonEpisodesState.Error -> item(key = "season-error") {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontalPadding, 24.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        Text(
                            text = stringResource(R.string.season_error),
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        TextButton(
                            onClick = onRetrySeason,
                            modifier = Modifier.focusRing(RoundedCornerShape(50))
                        ) {
                            Text(stringResource(R.string.retry))
                        }
                    }
                }
                is SeasonEpisodesState.Ready -> {
                    item(key = "season-header-${episodes.detail.season}") {
                        SeasonHeader(
                            season = seasonFor(state, episodes.detail.season),
                            detail = episodes.detail,
                            horizontalPadding = horizontalPadding,
                            onToggleSeasonWatched = onToggleSeasonWatched
                        )
                    }
                    itemsIndexed(episodes.detail.items, key = { _, episode -> episode.id }) { _, episode ->
                        EpisodeRow(
                            item = episode,
                            horizontalPadding = horizontalPadding,
                            onClick = {
                                onPlayEpisode(
                                    episode.id,
                                    episode.progress?.positionSeconds ?: 0.0
                                )
                            },
                            onToggleWatched = { onToggleEpisodeWatched(episode.id) }
                        )
                    }
                }
            }
        }

        if (!hasSeasons && state.similar.isNotEmpty()) {
            item(key = "tv-similar-title") {
                DetailSectionTitle(
                    title = stringResource(R.string.similar_section),
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp,
                        bottom = 8.dp
                    )
                )
            }
            item(key = "tv-similar-row") {
                TvSimilarSeriesRow(
                    similar = state.similar,
                    horizontalPadding = horizontalPadding,
                    firstFocusRequester = similarFocusRequester,
                    onOpenSeries = onOpenSeries
                )
            }
        }

        val cast = if (series.credits.cast.isNotEmpty()) {
            series.credits.cast
        } else {
            series.cast.map { name -> CastCredit(name = name, character = "", photoUrl = "") }
        }
        if (cast.isNotEmpty()) {
            item(key = "cast-title") {
                DetailSectionTitle(
                    title = stringResource(R.string.cast_section),
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp,
                        bottom = 8.dp
                    )
                )
            }
            item(key = "cast-row") {
                DetailCastRow(cast = cast, horizontalPadding = horizontalPadding)
            }
        }

        if (series.directors.isNotEmpty()) {
            item(key = "directors") {
                Column(
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp
                    ),
                    verticalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    DetailSectionTitle(title = stringResource(R.string.directors_section))
                    Text(
                        text = series.directors.joinToString(
                            stringResource(R.string.list_separator)
                        ),
                        style = MaterialTheme.typography.bodyLarge
                    )
                }
            }
        }

        if (hasSeasons && state.similar.isNotEmpty()) {
            item(key = "similar-title") {
                DetailSectionTitle(
                    title = stringResource(R.string.similar_section),
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 16.dp,
                        bottom = 8.dp
                    )
                )
            }
            item(key = "similar-row") {
                TvSimilarSeriesRow(
                    similar = state.similar,
                    horizontalPadding = horizontalPadding,
                    firstFocusRequester = null,
                    onOpenSeries = onOpenSeries
                )
            }
        }
    }
}

@Composable
private fun TvSimilarMoviesRow(
    similar: List<MediaItem>,
    horizontalPadding: Dp,
    firstFocusRequester: FocusRequester?,
    onOpenMovie: (Long) -> Unit
) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = horizontalPadding),
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        itemsIndexed(similar, key = { _, item -> item.id }) { index, item ->
            MediaPosterCard(
                item = item,
                width = 120.dp,
                onClick = { onOpenMovie(item.id) },
                modifier = if (index == 0 && firstFocusRequester != null) {
                    Modifier.focusRequester(firstFocusRequester)
                } else {
                    Modifier
                }
            )
        }
    }
}

@Composable
private fun TvSimilarSeriesRow(
    similar: List<Series>,
    horizontalPadding: Dp,
    firstFocusRequester: FocusRequester?,
    onOpenSeries: (Long) -> Unit
) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = horizontalPadding),
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        itemsIndexed(similar, key = { _, item -> item.id }) { index, item ->
            SeriesPosterCard(
                series = item,
                width = 120.dp,
                onClick = { onOpenSeries(item.id) },
                modifier = if (index == 0 && firstFocusRequester != null) {
                    Modifier.focusRequester(firstFocusRequester)
                } else {
                    Modifier
                }
            )
        }
    }
}

@Composable
private fun movieMetaLine(item: MediaItem): String {
    val separator = stringResource(R.string.list_separator)
    val genres = if (item.genres.isNotEmpty()) {
        item.genres.joinToString(separator)
    } else {
        null
    }
    val rating = item.rating?.let { stringResource(R.string.rating_format, it.toString()) }
    return listOfNotNull(
        item.year?.toString(),
        formatRuntime(item.runtimeMinutes).takeIf { it.isNotEmpty() },
        genres,
        rating
    ).joinToString(stringResource(R.string.meta_separator))
}

@Composable
private fun seriesMetaLine(series: SeriesDetail): String {
    val separator = stringResource(R.string.list_separator)
    val genres = if (series.genres.isNotEmpty()) {
        series.genres.joinToString(separator)
    } else {
        null
    }
    val counts = if (series.seasonCount > 0 || series.episodeCount > 0) {
        stringResource(R.string.series_counts, series.seasonCount, series.episodeCount)
    } else {
        null
    }
    val rating = series.rating?.let { stringResource(R.string.rating_format, it.toString()) }
    return listOfNotNull(
        series.year?.toString(),
        counts,
        genres,
        rating
    ).joinToString(stringResource(R.string.meta_separator))
}
