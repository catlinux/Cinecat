package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import io.github.catlinux.cinecat.core.api.CastCredit
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.tmdbImage
import kotlinx.coroutines.flow.collect

@Composable
fun MovieDetailScreen(
    viewModel: MovieDetailViewModel,
    onBack: () -> Unit,
    onPlay: (Long, Double) -> Unit,
    onOpenMovie: (Long) -> Unit,
    onOpenCollection: (Long) -> Unit
) {
    val state by viewModel.state.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }
    val actionError = stringResource(R.string.detail_action_error)
    val tablet = useNavigationRail(LocalConfiguration.current.screenWidthDp)
    val horizontalPadding = if (tablet) 28.dp else 16.dp

    ReloadOnResume(viewModel::load)

    LaunchedEffect(viewModel) {
        viewModel.events.collect {
            snackbarHostState.showSnackbar(actionError)
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        when (val current = state) {
            MovieDetailUiState.Loading -> CatalogLoading()
            MovieDetailUiState.Error -> CatalogError(
                message = stringResource(R.string.movie_detail_error),
                onRetry = viewModel::retry
            )
            is MovieDetailUiState.Ready -> MovieDetailContent(
                state = current,
                tablet = tablet,
                horizontalPadding = horizontalPadding,
                onPlay = onPlay,
                onOpenMovie = onOpenMovie,
                onOpenCollection = onOpenCollection,
                onToggleFavorite = viewModel::toggleFavorite,
                onToggleWatchlist = viewModel::toggleWatchlist,
                onToggleWatched = viewModel::toggleWatched
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
private fun MovieDetailContent(
    state: MovieDetailUiState.Ready,
    tablet: Boolean,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onPlay: (Long, Double) -> Unit,
    onOpenMovie: (Long) -> Unit,
    onOpenCollection: (Long) -> Unit,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit,
    onToggleWatched: () -> Unit
) {
    val item = state.item
    val isTelevision = LocalIsTelevision.current
    val playFocusRequester = remember(item.id) { FocusRequester() }
    val listState = rememberLazyListState()

    LaunchedEffect(isTelevision, item.id) {
        if (isTelevision) {
            // La lista no compone las acciones si quedan fuera de pantalla; las acercamos antes de pedir foco.
            runCatching { listState.scrollToItem(1) }
            withFrameNanos { }
            val focused = runCatching { playFocusRequester.requestFocus() }
            if (focused.isFailure) {
                runCatching { listState.scrollToItem(2) }
                withFrameNanos { }
                runCatching { playFocusRequester.requestFocus() }
            }
        }
    }

    LazyColumn(
        state = listState,
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(bottom = 24.dp)
    ) {
        item(key = "hero") {
            DetailBackdrop(
                title = item.title,
                url = tmdbImage(item.backdropUrl ?: item.posterUrl, "w1280"),
                height = if (tablet) 360.dp else 240.dp
            )
        }
        item(key = "info") {
            MovieInfoRow(
                item = item,
                horizontalPadding = horizontalPadding,
                tablet = tablet
            )
        }
        item(key = "actions") {
            MovieActions(
                item = item,
                horizontalPadding = horizontalPadding,
                playFocusRequester = playFocusRequester,
                onPlay = onPlay,
                onToggleFavorite = onToggleFavorite,
                onToggleWatchlist = onToggleWatchlist,
                onToggleWatched = onToggleWatched
            )
        }
        item(key = "synopsis") {
            DetailSynopsis(
                overview = item.overview,
                horizontalPadding = horizontalPadding
            )
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
                        top = 20.dp,
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
                        top = 20.dp
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
        val collection = item.collections.firstOrNull()
        if (collection != null) {
            item(key = "collection") {
                OutlinedButton(
                    onClick = { onOpenCollection(collection.id) },
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = horizontalPadding, end = horizontalPadding, top = 20.dp)
                        .focusRing(RoundedCornerShape(50))
                ) {
                    Text(stringResource(R.string.part_of_collection, collection.name))
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
                        MediaPosterCard(
                            item = similar,
                            width = 120.dp,
                            onClick = { onOpenMovie(similar.id) }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun MovieInfoRow(
    item: MediaItem,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    tablet: Boolean
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
            title = item.title,
            url = tmdbImage(item.posterUrl, "w342"),
            width = if (tablet) 180.dp else 128.dp
        )
        Column(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            Text(
                text = item.title,
                style = MaterialTheme.typography.headlineLarge,
                color = MaterialTheme.colorScheme.primary
            )
            val original = item.originalTitle?.takeIf {
                it.isNotBlank() && it != item.title
            }
            if (original != null) {
                Text(
                    text = original,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            item.year?.let { year ->
                DetailMetaText(year.toString())
            }
            val runtime = formatRuntime(item.runtimeMinutes)
            if (runtime.isNotEmpty()) {
                DetailMetaText(runtime)
            }
            if (item.genres.isNotEmpty()) {
                DetailMetaText(
                    item.genres.joinToString(stringResource(R.string.list_separator))
                )
            }
            item.rating?.let { rating ->
                Text(
                    text = stringResource(R.string.rating_format, rating.toString()),
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.secondary
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

@Composable
private fun MovieActions(
    item: MediaItem,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    playFocusRequester: FocusRequester,
    onPlay: (Long, Double) -> Unit,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit,
    onToggleWatched: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = horizontalPadding, end = horizontalPadding, top = 20.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        val progress = item.progress
        val position = progress?.positionSeconds?.takeIf { it > 0.0 }
        Button(
            onClick = { onPlay(item.id, position ?: 0.0) },
            modifier = Modifier
                .fillMaxWidth()
                .focusRequester(playFocusRequester)
                .focusRing(RoundedCornerShape(50))
        ) {
            Text(
                stringResource(
                    if (position != null) R.string.resume else R.string.play
                )
            )
        }
        if (position != null) {
            Text(
                text = stringResource(
                    R.string.resume_from,
                    formatPlaybackPosition(position)
                ),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            DetailActionChip(
                label = stringResource(R.string.favorite),
                active = item.favorite,
                icon = {
                    Icon(
                        imageVector = if (item.favorite) {
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
                active = item.watchlist,
                icon = {
                    Icon(
                        painter = painterResource(R.drawable.ic_watchlist),
                        contentDescription = null
                    )
                },
                onClick = onToggleWatchlist
            )
            DetailActionChip(
                label = stringResource(R.string.watched),
                active = item.watched,
                icon = {
                    Icon(
                        imageVector = Icons.Filled.CheckCircle,
                        contentDescription = null
                    )
                },
                onClick = onToggleWatched
            )
        }
    }
}
