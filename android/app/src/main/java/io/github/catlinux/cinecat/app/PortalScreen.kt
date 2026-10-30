package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountCircle
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import io.github.catlinux.cinecat.core.data.PortalEntry
import io.github.catlinux.cinecat.core.data.PortalRowData

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PortalScreen(
    viewModel: PortalViewModel,
    isTelevision: Boolean,
    onOpenAccount: (() -> Unit)?,
    onOpenMovie: (Long) -> Unit,
    onOpenSeries: (Long) -> Unit
) {
    val state by viewModel.state.collectAsState()
    val widthDp = LocalConfiguration.current.screenWidthDp
    val tablet = useNavigationRail(widthDp)
    val horizontalPadding = if (tablet) 28.dp else 16.dp
    val posterWidth = if (tablet) 150.dp else 104.dp
    val episodeWidth = if (tablet) 300.dp else 210.dp
    val refreshing = (state as? PortalUiState.Ready)?.refreshing == true

    // Sin pantalla táctil no hay gesto de tirar; el indicador solo se veía moverse al navegar.
    val refreshContainer: @Composable (@Composable () -> Unit) -> Unit = if (isTelevision) {
        { content -> Box(Modifier.fillMaxSize()) { content() } }
    } else {
        { content ->
            PullToRefreshBox(
                isRefreshing = refreshing,
                onRefresh = viewModel::refresh,
                modifier = Modifier.fillMaxSize()
            ) { content() }
        }
    }
    refreshContainer {
        Column(modifier = Modifier.fillMaxSize()) {
            PortalHeader(
                onOpenAccount = onOpenAccount,
                modifier = Modifier.padding(
                    start = horizontalPadding,
                    end = horizontalPadding,
                    top = 8.dp,
                    bottom = 4.dp
                )
            )
            LazyColumn(
                modifier = Modifier.weight(1f).fillMaxWidth(),
                contentPadding = PaddingValues(
                    start = horizontalPadding,
                    end = horizontalPadding,
                    top = 4.dp,
                    bottom = 24.dp
                )
            ) {
                when (val current = state) {
                    PortalUiState.Loading -> item(key = "portal-loading") {
                        PortalStatus {
                            CircularProgressIndicator()
                        }
                    }
                    is PortalUiState.Error -> item(key = "portal-error") {
                        PortalStatus {
                            Text(
                                text = stringResource(R.string.portal_error),
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            Button(onClick = viewModel::retry) {
                                Text(stringResource(R.string.retry))
                            }
                        }
                    }
                    PortalUiState.Empty -> item(key = "portal-empty") {
                        PortalStatus {
                            Text(
                                text = stringResource(R.string.portal_empty),
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                    is PortalUiState.Ready -> {
                        items(current.data.rows, key = { it.id }) { row ->
                            PortalRowSection(
                                row = row,
                                posterWidth = posterWidth,
                                episodeWidth = episodeWidth,
                                onOpenMovie = onOpenMovie,
                                onOpenSeries = onOpenSeries
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PortalHeader(
    onOpenAccount: (() -> Unit)?,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = stringResource(R.string.app_name),
            style = MaterialTheme.typography.displayLarge.copy(
                fontSize = 48.sp,
                lineHeight = 52.sp
            ),
            color = MaterialTheme.colorScheme.primary
        )
        Spacer(Modifier.weight(1f))
        if (onOpenAccount != null) {
            IconButton(onClick = onOpenAccount) {
                Icon(
                    imageVector = Icons.Filled.AccountCircle,
                    contentDescription = stringResource(R.string.account_title),
                    tint = MaterialTheme.colorScheme.primary
                )
            }
        }
    }
}

@Composable
private fun PortalRowSection(
    row: PortalRowData,
    posterWidth: Dp,
    episodeWidth: Dp,
    onOpenMovie: (Long) -> Unit,
    onOpenSeries: (Long) -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(bottom = 18.dp)
    ) {
        if (row.kicker.isNotBlank()) {
            Text(
                text = row.kicker.uppercase(),
                style = MaterialTheme.typography.labelSmall.copy(
                    fontWeight = FontWeight.Bold,
                    letterSpacing = 1.2.sp
                ),
                color = MaterialTheme.colorScheme.secondary
            )
        }
        Text(
            text = row.title,
            style = MaterialTheme.typography.headlineSmall,
            color = MaterialTheme.colorScheme.primary
        )
        Spacer(Modifier.height(10.dp))
        LazyRow(
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            items(
                items = row.entries,
                key = { entry ->
                    when (entry) {
                        is PortalEntry.Item -> "item-${entry.item.id}"
                        is PortalEntry.Series -> "series-${entry.series.id}"
                    }
                }
            ) { entry ->
                when (entry) {
                    is PortalEntry.Series -> SeriesPosterCard(
                        series = entry.series,
                        width = posterWidth,
                        onClick = { onOpenSeries(entry.series.id) }
                    )
                    is PortalEntry.Item -> {
                        val item = entry.item
                        if (item.kind == "episode") {
                            EpisodeCard(
                                item = item,
                                width = episodeWidth,
                                onClick = { onOpenSeries(item.seriesId ?: item.id) }
                            )
                        } else {
                            MediaPosterCard(
                                item = item,
                                width = posterWidth,
                                onClick = {
                                    if (item.kind == "series") {
                                        onOpenSeries(item.id)
                                    } else {
                                        onOpenMovie(item.id)
                                    }
                                }
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PortalStatus(content: @Composable () -> Unit) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = 240.dp),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            content()
        }
    }
}
