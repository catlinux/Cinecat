package io.github.catlinux.cinecat.app

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed as gridItemsIndexed
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.foundation.lazy.itemsIndexed as rowItemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedCard
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.Library
import io.github.catlinux.cinecat.core.api.PhotoItem
import kotlinx.coroutines.flow.collect

@Composable
fun PhotosScreen(
    viewModel: PhotosViewModel,
    isTelevision: Boolean
) {
    val state by viewModel.state.collectAsState()
    var viewerIndex by remember { mutableStateOf<Int?>(null) }
    val configuration = LocalConfiguration.current
    val tablet = useNavigationRail(configuration.screenWidthDp)
    val horizontalPadding = if (tablet) 28.dp else 16.dp
    val firstFocus = remember { FocusRequester() }

    LaunchedEffect(
        isTelevision,
        state.loadingLibraries,
        state.selectedLibrary?.id,
        state.folder,
        state.folders.size,
        state.paged.items.isNotEmpty(),
        state.libraries.size
    ) {
        if (!isTelevision || state.loadingLibraries) {
            return@LaunchedEffect
        }
        withFrameNanos { }
        val focused = runCatching { firstFocus.requestFocus() }
        if (focused.isFailure) {
            withFrameNanos { }
            runCatching { firstFocus.requestFocus() }
        }
    }

    BackHandler(enabled = viewerIndex == null && state.canGoUp) {
        viewModel.goUp()
    }

    Box(modifier = Modifier.fillMaxSize()) {
        when {
            state.loadingLibraries -> CatalogLoading()
            state.librariesError -> CatalogError(
                message = stringResource(R.string.photos_libraries_error),
                onRetry = viewModel::reloadLibraries
            )
            state.initialLoading -> CatalogLoading()
            state.showLibraries -> PhotoLibraries(
                libraries = state.libraries,
                horizontalPadding = horizontalPadding,
                firstFocus = firstFocus,
                isTelevision = isTelevision,
                onOpen = viewModel::selectLibrary
            )
            state.selectedLibrary != null -> PhotoFolder(
                state = state,
                viewModel = viewModel,
                horizontalPadding = horizontalPadding,
                firstFocus = firstFocus,
                isTelevision = isTelevision,
                onOpenPhoto = { index -> viewerIndex = index }
            )
            else -> CatalogEmpty(stringResource(R.string.photos_libraries_empty))
        }
    }

    val currentViewerIndex = viewerIndex
    if (currentViewerIndex != null) {
        PhotoViewerScreen(
            photos = state.paged.items,
            initialIndex = currentViewerIndex,
            isTelevision = isTelevision,
            onClose = { viewerIndex = null }
        )
    }
}

@Composable
private fun PhotoLibraries(
    libraries: List<Library>,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    firstFocus: FocusRequester,
    isTelevision: Boolean,
    onOpen: (Library) -> Unit
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.padding(
                start = horizontalPadding,
                end = horizontalPadding,
                top = 16.dp
            )
        ) {
            Text(
                text = stringResource(R.string.nav_photos),
                style = MaterialTheme.typography.headlineLarge,
                color = MaterialTheme.colorScheme.primary
            )
            if (libraries.isNotEmpty()) {
                Text(
                    text = pluralStringResource(
                        R.plurals.photos_library_count,
                        libraries.size,
                        libraries.size
                    ),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
        if (libraries.isEmpty()) {
            Box(modifier = Modifier.weight(1f)) {
                CatalogEmpty(stringResource(R.string.photos_libraries_empty))
            }
        } else {
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 180.dp),
                contentPadding = PaddingValues(
                    start = horizontalPadding,
                    end = horizontalPadding,
                    top = 16.dp,
                    bottom = 24.dp
                ),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                modifier = Modifier.weight(1f).fillMaxWidth()
            ) {
                gridItemsIndexed(libraries, key = { _, library -> library.id }) { index, library ->
                    PhotoLibraryCard(
                        library = library,
                        focusRequester = if (isTelevision && index == 0) firstFocus else null,
                        onClick = { onOpen(library) }
                    )
                }
            }
        }
    }
}

@Composable
private fun PhotoLibraryCard(
    library: Library,
    focusRequester: FocusRequester?,
    onClick: () -> Unit
) {
    val modifier = Modifier
        .fillMaxWidth()
        .then(if (focusRequester != null) Modifier.focusRequester(focusRequester) else Modifier)
        .clickable(onClick = onClick)
        .tvFocusPadding()
    OutlinedCard(
        modifier = modifier,
        colors = CardDefaults.outlinedCardColors(
            containerColor = MaterialTheme.colorScheme.surface
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Text(
                text = library.name,
                style = MaterialTheme.typography.titleLarge,
                color = MaterialTheme.colorScheme.primary
            )
            Text(
                text = pluralStringResource(
                    R.plurals.photos_count,
                    library.itemCount.coerceAtMost(Int.MAX_VALUE.toLong()).toInt(),
                    library.itemCount
                ),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun PhotoFolder(
    state: PhotosUiState,
    viewModel: PhotosViewModel,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    firstFocus: FocusRequester,
    isTelevision: Boolean,
    onOpenPhoto: (Int) -> Unit
) {
    val library = state.selectedLibrary ?: return
    val title = state.folder.substringAfterLast('/').ifBlank { library.name }
    val countLabel = pluralStringResource(
        R.plurals.photos_count,
        state.paged.total.coerceAtMost(Int.MAX_VALUE.toLong()).toInt(),
        state.paged.total
    )
    val sortLabel = when (state.sort) {
        PhotoSort.Date -> stringResource(R.string.photos_sort_date)
        PhotoSort.Name -> stringResource(R.string.photos_sort_name)
    }
    val sortOptions = listOf(
        FilterOption(PhotoSort.Date.apiValue, stringResource(R.string.photos_sort_date)),
        FilterOption(PhotoSort.Name.apiValue, stringResource(R.string.photos_sort_name))
    )

    Column(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.padding(
                start = horizontalPadding,
                end = horizontalPadding,
                top = 12.dp
            )
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                if (state.canGoUp) {
                    TextButton(
                        onClick = { viewModel.goUp() },
                        modifier = Modifier.focusRing(RoundedCornerShape(8.dp))
                    ) {
                        Text(stringResource(R.string.photos_up))
                    }
                }
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = library.name,
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Text(
                        text = title,
                        style = MaterialTheme.typography.headlineLarge,
                        color = MaterialTheme.colorScheme.primary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
                Text(
                    text = countLabel,
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                FilterMenu(
                    label = stringResource(R.string.photos_sort),
                    selectedLabel = sortLabel,
                    options = sortOptions,
                    onSelected = { value -> viewModel.setSort(PhotoSort.fromApi(value)) }
                )
            }
        }

        if (state.folders.isNotEmpty()) {
            Text(
                text = stringResource(R.string.photos_folders),
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.padding(
                    start = horizontalPadding,
                    end = horizontalPadding,
                    top = 12.dp
                )
            )
            LazyRow(
                contentPadding = PaddingValues(horizontal = horizontalPadding, vertical = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                rowItemsIndexed(state.folders, key = { _, folder -> folder }) { index, folder ->
                    PhotoFolderCard(
                        name = folder,
                        focusRequester = if (isTelevision && index == 0) firstFocus else null,
                        onClick = {
                            viewModel.openFolder(childFolder(state.folder, folder))
                        }
                    )
                }
            }
        }

        Box(modifier = Modifier.weight(1f).fillMaxWidth()) {
            when {
                state.paged.items.isEmpty() && state.paged.error -> CatalogError(
                    message = stringResource(R.string.photos_error),
                    onRetry = viewModel::retry
                )
                state.paged.items.isEmpty() && state.initialLoading -> CatalogLoading()
                state.paged.items.isEmpty() && state.folders.isEmpty() ->
                    CatalogEmpty(stringResource(R.string.photos_empty))
                state.paged.items.isEmpty() -> Unit
                else -> PhotoGrid(
                    state = state,
                    horizontalPadding = horizontalPadding,
                    firstFocus = firstFocus,
                    isTelevision = isTelevision,
                    focusFirstPhoto = state.folders.isEmpty(),
                    onOpenPhoto = onOpenPhoto,
                    onRetry = viewModel::retry,
                    loadMore = viewModel::loadMore
                )
            }
        }
    }
}

@Composable
private fun PhotoFolderCard(
    name: String,
    focusRequester: FocusRequester?,
    onClick: () -> Unit
) {
    val modifier = Modifier
        .width(170.dp)
        .then(if (focusRequester != null) Modifier.focusRequester(focusRequester) else Modifier)
        .clickable(onClick = onClick)
        .tvFocusPadding()
    OutlinedCard(
        modifier = modifier,
        colors = CardDefaults.outlinedCardColors(
            containerColor = MaterialTheme.colorScheme.surface
        )
    ) {
        Column(
            modifier = Modifier.padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Text(
                text = name,
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.primary,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis
            )
            Text(
                text = stringResource(R.string.photos_open_folder),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun PhotoGrid(
    state: PhotosUiState,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    firstFocus: FocusRequester,
    isTelevision: Boolean,
    focusFirstPhoto: Boolean,
    onOpenPhoto: (Int) -> Unit,
    onRetry: () -> Unit,
    loadMore: () -> Unit
) {
    val gridState = rememberLazyGridState()
    LaunchedEffect(gridState, state.paged.items.size, state.paged.hasMore, state.paged.error) {
        if (state.paged.items.isNotEmpty() && state.paged.hasMore && !state.paged.error) {
            snapshotFlow { gridState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1 }
                .collect { lastVisible ->
                    if (lastVisible >= state.paged.items.size - 6) {
                        loadMore()
                    }
                }
        }
    }
    PhotoGridContent(
        state = state,
        gridState = gridState,
        horizontalPadding = horizontalPadding,
        firstFocus = firstFocus,
        isTelevision = isTelevision,
        focusFirstPhoto = focusFirstPhoto,
        onOpenPhoto = onOpenPhoto,
        onRetry = onRetry
    )
}

@Composable
private fun PhotoGridContent(
    state: PhotosUiState,
    gridState: LazyGridState,
    horizontalPadding: androidx.compose.ui.unit.Dp,
    firstFocus: FocusRequester,
    isTelevision: Boolean,
    focusFirstPhoto: Boolean,
    onOpenPhoto: (Int) -> Unit,
    onRetry: () -> Unit
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 120.dp),
        state = gridState,
        contentPadding = PaddingValues(
            start = horizontalPadding,
            end = horizontalPadding,
            top = 4.dp,
            bottom = 24.dp
        ),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxSize()
    ) {
        gridItemsIndexed(state.paged.items, key = { _, photo -> photo.id }) { index, photo ->
            PhotoTile(
                photo = photo,
                focusRequester = if (isTelevision && focusFirstPhoto && index == 0) {
                    firstFocus
                } else {
                    null
                },
                onClick = { onOpenPhoto(index) }
            )
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
private fun PhotoTile(
    photo: PhotoItem,
    focusRequester: FocusRequester?,
    onClick: () -> Unit
) {
    val date = photoDateLabel(photo.takenAt)
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (focusRequester != null) Modifier.focusRequester(focusRequester) else Modifier)
            .clickable(onClick = onClick)
            .tvFocusPadding(),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        AsyncImage(
            model = photoThumbPath(photo.id),
            contentDescription = stringResource(R.string.photo_description, photo.name),
            contentScale = ContentScale.Crop,
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(photoAspectRatio(photo.width, photo.height))
                .clip(RoundedCornerShape(4.dp))
                .background(MaterialTheme.colorScheme.surfaceVariant)
        )
        Text(
            text = photo.name,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurface,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis
        )
        if (date.isNotEmpty()) {
            Text(
                text = date,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}
