package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.CollectionDetail as ApiCollectionDetail
import io.github.catlinux.cinecat.core.api.tmdbImage
import kotlinx.coroutines.flow.collect

@Composable
fun CollectionDetailScreen(
    viewModel: CollectionDetailViewModel,
    onBack: () -> Unit,
    onOpenMovie: (Long) -> Unit
) {
    val state by viewModel.state.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }
    val actionError = stringResource(R.string.detail_action_error)
    val tablet = useNavigationRail(LocalConfiguration.current.screenWidthDp)
    val horizontalPadding = if (tablet) 28.dp else 16.dp

    LaunchedEffect(viewModel) {
        viewModel.events.collect {
            snackbarHostState.showSnackbar(actionError)
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        when (val current = state) {
            CollectionDetailUiState.Loading -> CatalogLoading()
            CollectionDetailUiState.Error -> CatalogError(
                message = stringResource(R.string.collection_detail_error),
                onRetry = viewModel::retry
            )
            is CollectionDetailUiState.Ready -> CollectionDetailContent(
                collection = current.collection,
                tablet = tablet,
                horizontalPadding = horizontalPadding,
                onOpenMovie = onOpenMovie,
                onToggleFavorite = viewModel::toggleFavorite,
                onToggleWatchlist = viewModel::toggleWatchlist
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
private fun CollectionDetailContent(
    collection: ApiCollectionDetail,
    tablet: Boolean,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onOpenMovie: (Long) -> Unit,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 110.dp),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(
            start = horizontalPadding,
            end = horizontalPadding,
            bottom = 24.dp
        ),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        item(span = { GridItemSpan(maxLineSpan) }) {
            CollectionHero(
                collection = collection,
                height = if (tablet) 360.dp else 240.dp
            )
        }
        item(span = { GridItemSpan(maxLineSpan) }) {
            CollectionInfo(
                collection = collection,
                onToggleFavorite = onToggleFavorite,
                onToggleWatchlist = onToggleWatchlist
            )
        }
        if (collection.items.isEmpty()) {
            item(span = { GridItemSpan(maxLineSpan) }) {
                Text(
                    text = stringResource(R.string.collection_empty),
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(vertical = 24.dp)
                )
            }
        } else {
            items(collection.items, key = { it.id }) { item ->
                BoxWithConstraints {
                    MediaPosterCard(
                        item = item,
                        width = maxWidth,
                        onClick = { onOpenMovie(item.id) }
                    )
                }
            }
        }
    }
}

@Composable
private fun CollectionHero(collection: ApiCollectionDetail, height: androidx.compose.ui.unit.Dp) {
    val firstItem = collection.items.firstOrNull()
    val imageUrl = tmdbImage(
        collection.backdropUrl
            ?: firstItem?.backdropUrl
            ?: collection.posterUrl
            ?: firstItem?.posterUrl,
        "w1280"
    )
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(height)
            .background(MaterialTheme.colorScheme.surface)
    ) {
        if (imageUrl != null) {
            AsyncImage(
                model = imageUrl,
                contentDescription = stringResource(
                    R.string.backdrop_description,
                    collection.name
                ),
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize()
            )
        }
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    Brush.verticalGradient(
                        colors = listOf(
                            Color.Transparent,
                            Color.Transparent,
                            MaterialTheme.colorScheme.background
                        )
                    )
                )
        )
    }
}

@Composable
private fun CollectionInfo(
    collection: ApiCollectionDetail,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            text = collection.name,
            style = MaterialTheme.typography.headlineLarge,
            color = MaterialTheme.colorScheme.primary
        )
        val overview = collection.overview?.trim().orEmpty()
        if (overview.isNotEmpty()) {
            Text(
                text = overview,
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        Text(
            text = pluralStringResource(
                R.plurals.collection_item_count,
                collection.itemCount.toInt(),
                collection.itemCount
            ),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            DetailActionChip(
                label = stringResource(R.string.favorite),
                active = collection.favorite,
                icon = {
                    Icon(
                        imageVector = if (collection.favorite) {
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
                active = collection.watchlist,
                icon = {
                    Icon(
                        painter = painterResource(R.drawable.ic_watchlist),
                        contentDescription = null
                    )
                },
                onClick = onToggleWatchlist
            )
        }
    }
}
