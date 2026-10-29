package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PrimaryScrollableTabRow
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.CastCredit
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.Season
import io.github.catlinux.cinecat.core.api.SeriesDetail
import io.github.catlinux.cinecat.core.api.tmdbImage
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch

@Composable
fun SeriesDetailScreen(
    viewModel: SeriesDetailViewModel,
    onBack: () -> Unit,
    onOpenSeries: (Long) -> Unit
) {
    val state by viewModel.state.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()
    val actionError = stringResource(R.string.detail_action_error)
    val playSoon = stringResource(R.string.play_soon_snackbar)
    val tablet = useNavigationRail(LocalConfiguration.current.screenWidthDp)
    val horizontalPadding = if (tablet) 28.dp else 16.dp

    LaunchedEffect(viewModel) {
        viewModel.events.collect {
            snackbarHostState.showSnackbar(actionError)
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        when (val current = state) {
            SeriesDetailUiState.Loading -> CatalogLoading()
            SeriesDetailUiState.Error -> CatalogError(
                message = stringResource(R.string.series_detail_error),
                onRetry = viewModel::retry
            )
            is SeriesDetailUiState.Ready -> SeriesDetailContent(
                state = current,
                tablet = tablet,
                horizontalPadding = horizontalPadding,
                onSelectSeason = viewModel::selectSeason,
                onRetrySeason = viewModel::retrySeason,
                onToggleFavorite = viewModel::toggleFavorite,
                onToggleWatchlist = viewModel::toggleWatchlist,
                onToggleSeriesWatched = viewModel::toggleSeriesWatched,
                onToggleSeasonWatched = viewModel::toggleSeasonWatched,
                onToggleEpisodeWatched = viewModel::toggleEpisodeWatched,
                onEpisodeClick = {
                    scope.launch { snackbarHostState.showSnackbar(playSoon) }
                },
                onOpenSeries = onOpenSeries
            )
        }
        DetailBackButton(
            onBack = onBack,
            modifier = Modifier
                .align(Alignment.TopStart)
                .padding(16.dp)
        )
        SnackbarHost(
            hostState = snackbarHostState,
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .padding(16.dp)
        )
    }
}

@Composable
private fun SeriesDetailContent(
    state: SeriesDetailUiState.Ready,
    tablet: Boolean,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onSelectSeason: (Int) -> Unit,
    onRetrySeason: () -> Unit,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit,
    onToggleSeriesWatched: () -> Unit,
    onToggleSeasonWatched: () -> Unit,
    onToggleEpisodeWatched: (Long) -> Unit,
    onEpisodeClick: () -> Unit,
    onOpenSeries: (Long) -> Unit
) {
    val series = state.series
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(bottom = 24.dp)
    ) {
        item(key = "hero") {
            DetailBackdrop(
                title = series.title,
                url = tmdbImage(series.backdropUrl ?: series.posterUrl, "w1280"),
                height = if (tablet) 360.dp else 240.dp
            )
        }
        item(key = "info") {
            SeriesInfoRow(
                series = series,
                tablet = tablet,
                horizontalPadding = horizontalPadding
            )
        }
        item(key = "actions") {
            SeriesActions(
                state = state,
                horizontalPadding = horizontalPadding,
                onToggleFavorite = onToggleFavorite,
                onToggleWatchlist = onToggleWatchlist,
                onToggleSeriesWatched = onToggleSeriesWatched
            )
        }
        item(key = "synopsis") {
            DetailSynopsis(
                overview = series.overview,
                horizontalPadding = horizontalPadding
            )
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
                        top = 20.dp,
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
                        top = 20.dp
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
        if (state.seasons.isNotEmpty()) {
            item(key = "season-tabs") {
                SeasonTabs(
                    seasons = state.seasons,
                    selectedSeason = state.selectedSeason,
                    horizontalPadding = horizontalPadding,
                    onSelectSeason = onSelectSeason
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
                        TextButton(onClick = onRetrySeason) {
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
                    items(episodes.detail.items, key = { it.id }) { episode ->
                        EpisodeRow(
                            item = episode,
                            horizontalPadding = horizontalPadding,
                            onClick = onEpisodeClick,
                            onToggleWatched = { onToggleEpisodeWatched(episode.id) }
                        )
                    }
                }
            }
        }
        if (state.similar.isNotEmpty()) {
            item(key = "similar-title") {
                DetailSectionTitle(
                    title = stringResource(R.string.similar_section),
                    modifier = Modifier.padding(
                        start = horizontalPadding,
                        end = horizontalPadding,
                        top = 20.dp,
                        bottom = 8.dp
                    )
                )
            }
            item(key = "similar-row") {
                LazyRow(
                    contentPadding = PaddingValues(horizontal = horizontalPadding),
                    horizontalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    items(state.similar, key = { it.id }) { similar ->
                        SeriesPosterCard(
                            series = similar,
                            width = 120.dp,
                            onClick = { onOpenSeries(similar.id) }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SeriesInfoRow(
    series: SeriesDetail,
    tablet: Boolean,
    horizontalPadding: androidx.compose.ui.unit.Dp
) {
    Row(
        modifier = Modifier.padding(
            start = horizontalPadding,
            end = horizontalPadding,
            top = 16.dp
        ),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalAlignment = Alignment.Top
    ) {
        DetailPoster(
            title = series.title,
            url = tmdbImage(series.posterUrl, "w342"),
            width = if (tablet) 180.dp else 128.dp
        )
        Column(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            Text(
                text = series.title,
                style = MaterialTheme.typography.headlineLarge,
                color = MaterialTheme.colorScheme.primary
            )
            val original = series.originalTitle?.takeIf {
                it.isNotBlank() && it != series.title
            }
            if (original != null) {
                Text(
                    text = original,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            series.year?.let { DetailMetaText(it.toString()) }
            if (series.genres.isNotEmpty()) {
                DetailMetaText(
                    series.genres.joinToString(stringResource(R.string.list_separator))
                )
            }
            series.rating?.let { rating ->
                Text(
                    text = stringResource(R.string.rating_format, rating.toString()),
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.secondary
                )
            }
            if (series.seasonCount > 0 || series.episodeCount > 0) {
                DetailMetaText(
                    stringResource(
                        R.string.series_counts,
                        series.seasonCount,
                        series.episodeCount
                    )
                )
            }
        }
    }
}

@Composable
private fun SeriesActions(
    state: SeriesDetailUiState.Ready,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit,
    onToggleSeriesWatched: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(start = horizontalPadding, end = horizontalPadding, top = 20.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        DetailActionChip(
            label = stringResource(R.string.favorite),
            active = state.series.favorite,
            icon = {
                Icon(
                    imageVector = if (state.series.favorite) {
                        Icons.Filled.Favorite
                    } else {
                        Icons.Filled.FavoriteBorder
                    },
                    contentDescription = null
                )
            },
            onClick = onToggleFavorite
        )
        DetailActionChip(
            label = stringResource(R.string.watchlist),
            active = state.series.watchlist,
            icon = {
                Icon(
                    painter = painterResource(R.drawable.ic_watchlist),
                    contentDescription = null
                )
            },
            onClick = onToggleWatchlist
        )
        DetailActionChip(
            label = stringResource(R.string.series_watched),
            active = state.seriesWatched,
            icon = {
                Icon(
                    imageVector = Icons.Filled.CheckCircle,
                    contentDescription = null
                )
            },
            onClick = onToggleSeriesWatched
        )
    }
}

@Composable
private fun SeasonTabs(
    seasons: List<Season>,
    selectedSeason: Int?,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onSelectSeason: (Int) -> Unit
) {
    val selectedIndex = seasons.indexOfFirst { it.season == selectedSeason }
        .coerceAtLeast(0)
    val labels = seasons.map { season ->
        if (season.season == 0) {
            stringResource(R.string.season_specials)
        } else {
            stringResource(R.string.season_number, season.season)
        }
    }
    PrimaryScrollableTabRow(
        selectedTabIndex = selectedIndex,
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 20.dp),
        edgePadding = horizontalPadding,
        containerColor = MaterialTheme.colorScheme.background,
        contentColor = MaterialTheme.colorScheme.primary
    ) {
        seasons.forEachIndexed { index, season ->
            Tab(
                selected = index == selectedIndex,
                onClick = { onSelectSeason(season.season) },
                text = {
                    Text(
                        text = labels[index],
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            )
        }
    }
}

@Composable
private fun SeasonHeader(
    season: Season?,
    detail: io.github.catlinux.cinecat.core.api.SeasonDetail,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onToggleSeasonWatched: () -> Unit
) {
    val watched = season != null &&
        season.episodeCount > 0 &&
        season.watchedCount >= season.episodeCount
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = horizontalPadding, end = horizontalPadding, top = 16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = if (detail.season == 0) {
                stringResource(R.string.season_specials)
            } else {
                stringResource(R.string.season_number, detail.season)
            },
            style = MaterialTheme.typography.headlineSmall,
            color = MaterialTheme.colorScheme.primary
        )
        if (season != null) {
            Text(
                text = stringResource(
                    R.string.season_progress,
                    season.watchedCount,
                    season.episodeCount
                ),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        DetailActionChip(
            label = stringResource(R.string.season_watched),
            active = watched,
            icon = {
                Icon(
                    imageVector = Icons.Filled.CheckCircle,
                    contentDescription = null
                )
            },
            onClick = onToggleSeasonWatched
        )
    }
}

@Composable
private fun EpisodeRow(
    item: MediaItem,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onClick: () -> Unit,
    onToggleWatched: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = horizontalPadding, vertical = 10.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.Top
    ) {
        val image = tmdbImage(
            item.stillUrl ?: item.seriesBackdropUrl ?: item.backdropUrl,
            "w342"
        )
        Box(
            modifier = Modifier
                .width(128.dp)
                .aspectRatio(16f / 9f)
                .clip(RoundedCornerShape(4.dp))
                .background(MaterialTheme.colorScheme.surface),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = formatEpisodeCode(item.season, item.episode),
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onSurface
            )
            if (image != null) {
                AsyncImage(
                    model = image,
                    contentDescription = stringResource(
                        R.string.episode_image_description,
                        item.title
                    ),
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize()
                )
            }
        }
        Column(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Row(verticalAlignment = Alignment.Top) {
                Column(modifier = Modifier.weight(1f)) {
                    val code = formatEpisodeCode(item.season, item.episode)
                    if (code.isNotEmpty()) {
                        Text(
                            text = code,
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.secondary
                        )
                    }
                    Text(
                        text = item.episodeTitle ?: item.title,
                        style = MaterialTheme.typography.titleMedium,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                }
                IconButton(onClick = onToggleWatched) {
                    Icon(
                        imageVector = Icons.Filled.CheckCircle,
                        contentDescription = stringResource(
                            if (item.watched) R.string.unmark_watched else R.string.mark_watched
                        ),
                        tint = if (item.watched) {
                            MaterialTheme.colorScheme.primary
                        } else {
                            MaterialTheme.colorScheme.onSurfaceVariant
                        }
                    )
                }
            }
            val runtime = formatRuntime(item.runtimeMinutes)
            if (runtime.isNotEmpty()) {
                Text(
                    text = runtime,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            item.overview?.let { overview ->
                Text(
                    text = overview,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis
                )
            }
            item.progress?.let { progress ->
                val fraction = if (progress.durationSeconds > 0.0) {
                    (progress.positionSeconds / progress.durationSeconds)
                        .coerceIn(0.0, 1.0)
                        .toFloat()
                } else {
                    0f
                }
                if (fraction > 0f) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(4.dp)
                            .background(MaterialTheme.colorScheme.surfaceVariant)
                    ) {
                        Box(
                            modifier = Modifier
                                .fillMaxWidth(fraction)
                                .fillMaxHeight()
                                .background(MaterialTheme.colorScheme.primary)
                        )
                    }
                }
            }
        }
    }
}

private fun seasonFor(
    state: SeriesDetailUiState.Ready,
    seasonNumber: Int
): Season? = state.seasons.firstOrNull { it.season == seasonNumber }
