package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.collect

@Composable
fun MoviesScreen(
    viewModel: MoviesViewModel,
    onOpenMovie: (Long) -> Unit
) {
    val state by viewModel.state.collectAsState()
    val gridState = rememberLazyGridState()
    val configuration = LocalConfiguration.current
    val tablet = useNavigationRail(configuration.screenWidthDp)
    val mobile = !tablet && !LocalIsTelevision.current &&
        configuration.screenWidthDp < configuration.screenHeightDp
    val horizontalPadding = if (tablet) 28.dp else 16.dp
    val sortOptions = movieSortOptions()
    val allGenres = stringResource(R.string.filter_all)
    val genreOptions = listOf(FilterOption(null, allGenres)) +
        state.genres.sortedBy { it.lowercase() }.map { FilterOption(it, it) }
    val sortLabel = sortOptions.firstOrNull { it.value == state.sort }?.label
        ?: stringResource(R.string.sort_title)
    val genreLabel = state.genre ?: allGenres

    LaunchedEffect(gridState, state.paged.items.size, state.paged.hasMore) {
        if (state.paged.items.isNotEmpty() && state.paged.hasMore) {
            snapshotFlow { gridState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1 }
                .collect { lastVisible ->
                    if (lastVisible >= state.paged.items.size - 12) {
                        viewModel.loadMore()
                    }
                }
        }
    }
    LaunchedEffect(state.sort, state.genre) {
        gridState.scrollToItem(0)
    }

    Column(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.padding(
                start = horizontalPadding,
                end = horizontalPadding,
                top = 16.dp
            )
        ) {
            Text(
                text = stringResource(R.string.nav_movies),
                style = MaterialTheme.typography.headlineLarge,
                color = MaterialTheme.colorScheme.primary
            )
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState())
                    .padding(top = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                FilterMenu(
                    label = stringResource(R.string.filter_sort),
                    selectedLabel = sortLabel,
                    options = sortOptions,
                    onSelected = { value -> value?.let(viewModel::setSort) }
                )
                FilterMenu(
                    label = stringResource(R.string.filter_genre),
                    selectedLabel = genreLabel,
                    options = genreOptions,
                    onSelected = viewModel::setGenre
                )
            }
        }
        Box(modifier = Modifier.weight(1f).fillMaxWidth()) {
            when {
                state.initialLoading -> CatalogLoading()
                state.paged.items.isEmpty() && state.paged.error -> CatalogError(
                    message = stringResource(R.string.movies_error),
                    onRetry = viewModel::retry
                )
                state.isEmpty -> CatalogEmpty(stringResource(R.string.movies_empty))
                else -> MovieGrid(
                    state = state,
                    gridState = gridState,
                    horizontalPadding = horizontalPadding,
                    onOpenMovie = onOpenMovie,
                    onRetry = viewModel::retry,
                    ensureLoaded = viewModel::ensureLoaded,
                    mobile = mobile
                )
            }
        }
    }
}

@Composable
private fun MovieGrid(
    state: MoviesUiState,
    gridState: LazyGridState,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onOpenMovie: (Long) -> Unit,
    onRetry: () -> Unit,
    ensureLoaded: suspend (Int) -> Unit,
    mobile: Boolean
) {
    Row(modifier = Modifier.fillMaxSize()) {
        LazyVerticalGrid(
            columns = if (mobile) GridCells.Fixed(4) else GridCells.Adaptive(minSize = 110.dp),
            state = gridState,
            modifier = Modifier
                .weight(1f)
                .fillMaxHeight(),
            contentPadding = PaddingValues(
                start = horizontalPadding,
                end = horizontalPadding,
                top = 12.dp,
                bottom = 24.dp
            ),
            horizontalArrangement = Arrangement.spacedBy(if (mobile) 8.dp else 12.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            items(state.paged.items, key = { it.id }) { item ->
                BoxWithConstraints {
                    MediaPosterCard(
                        item = item,
                        width = maxWidth,
                        onClick = { onOpenMovie(item.id) },
                        compact = mobile
                    )
                }
            }
            if (state.paged.loading && state.paged.items.isNotEmpty()) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    CatalogMoreLoading()
                }
            }
            if (state.paged.error && state.paged.items.isNotEmpty()) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    CatalogMoreError(onRetry = onRetry)
                }
            }
        }
        TvLetterIndex(
            letters = state.letters,
            gridState = gridState,
            ensureLoaded = ensureLoaded
        )
    }
}

@Composable
internal fun CatalogLoading() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        CircularProgressIndicator()
    }
}

@Composable
internal fun CatalogError(message: String, onRetry: () -> Unit) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = message,
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Button(onClick = onRetry) {
                Text(stringResource(R.string.retry))
            }
        }
    }
}

@Composable
internal fun CatalogEmpty(message: String) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = message,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}

@Composable
internal fun CatalogMoreLoading() {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp),
        contentAlignment = Alignment.Center
    ) {
        CircularProgressIndicator()
    }
}

@Composable
internal fun CatalogMoreError(onRetry: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Button(onClick = onRetry) {
            Text(stringResource(R.string.retry))
        }
    }
}

@Composable
private fun movieSortOptions(): List<FilterOption> = listOf(
    FilterOption("title", stringResource(R.string.sort_title)),
    FilterOption("year", stringResource(R.string.sort_year)),
    FilterOption("added", stringResource(R.string.sort_added))
)
