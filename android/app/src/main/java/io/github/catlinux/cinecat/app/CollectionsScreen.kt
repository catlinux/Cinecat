package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.collect

@Composable
fun CollectionsScreen(
    viewModel: CollectionsViewModel,
    onOpenCollection: (Long) -> Unit
) {
    val state by viewModel.state.collectAsState()
    val gridState = rememberLazyGridState()
    val tablet = useNavigationRail(LocalConfiguration.current.screenWidthDp)
    val horizontalPadding = if (tablet) 28.dp else 16.dp
    val sortOptions = collectionSortOptions()
    val sortLabel = sortOptions.firstOrNull { it.value == state.sort }?.label
        ?: stringResource(R.string.sort_name)

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
    LaunchedEffect(state.sort) {
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
                text = stringResource(R.string.nav_collections),
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
            }
        }
        Box(modifier = Modifier.weight(1f).fillMaxWidth()) {
            when {
                state.initialLoading -> CatalogLoading()
                state.paged.items.isEmpty() && state.paged.error -> CatalogError(
                    message = stringResource(R.string.collections_error),
                    onRetry = viewModel::retry
                )
                state.isEmpty -> CatalogEmpty(stringResource(R.string.collections_empty))
                else -> CollectionsGrid(
                    state = state,
                    gridState = gridState,
                    horizontalPadding = horizontalPadding,
                    onOpenCollection = onOpenCollection,
                    onRetry = viewModel::retry
                )
            }
        }
    }
}

@Composable
private fun CollectionsGrid(
    state: CollectionsUiState,
    gridState: LazyGridState,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    onOpenCollection: (Long) -> Unit,
    onRetry: () -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 150.dp),
        state = gridState,
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(
            start = horizontalPadding,
            end = horizontalPadding,
            top = 12.dp,
            bottom = 24.dp
        ),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        items(state.paged.items, key = { it.id }) { collection ->
            BoxWithConstraints {
                CollectionPosterCard(
                    collection = collection,
                    width = maxWidth,
                    onClick = { onOpenCollection(collection.id) }
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
}

@Composable
private fun collectionSortOptions(): List<FilterOption> = listOf(
    FilterOption("name", stringResource(R.string.sort_name)),
    FilterOption("added", stringResource(R.string.sort_added)),
    FilterOption("size", stringResource(R.string.sort_size))
)
