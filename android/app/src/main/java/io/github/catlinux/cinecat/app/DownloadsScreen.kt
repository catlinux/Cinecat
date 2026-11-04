package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.DownloadRecord
import io.github.catlinux.cinecat.core.api.tmdbImage
import kotlinx.coroutines.launch

@Composable
fun DownloadsScreen(
    downloads: DownloadRepository,
    onPlay: (DownloadRecord) -> Unit,
    onBack: () -> Unit
) {
    if (LocalIsTelevision.current) {
        return
    }
    val state by downloads.records.collectAsState()
    val statuses by downloads.statuses.collectAsState()
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        downloads.flushPending()
    }

    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            TextButton(onClick = onBack) {
                Text(stringResource(R.string.back))
            }
            Text(
                text = stringResource(R.string.downloads_title),
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.padding(start = 12.dp)
            )
        }
        if (state.downloads.isEmpty()) {
            Box(
                modifier = Modifier.fillMaxSize(),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = stringResource(R.string.downloads_empty),
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        } else {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(
                    start = 16.dp,
                    end = 16.dp,
                    top = 8.dp,
                    bottom = 24.dp
                ),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                items(state.downloads, key = { it.itemId }) { record ->
                    DownloadRow(
                        record = record,
                        status = statuses[record.itemId],
                        onPlay = { onPlay(record) },
                        onCancel = { scope.launch { downloads.cancel(record) } },
                        onDelete = { scope.launch { downloads.delete(record) } }
                    )
                }
            }
        }
    }
}

@Composable
private fun DownloadRow(
    record: DownloadRecord,
    status: DownloadStatus?,
    onPlay: () -> Unit,
    onCancel: () -> Unit,
    onDelete: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(10.dp))
            .padding(10.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.Top
    ) {
        Box(
            modifier = Modifier
                .width(72.dp)
                .height(108.dp)
                .clip(RoundedCornerShape(6.dp))
                .background(MaterialTheme.colorScheme.surfaceVariant),
            contentAlignment = Alignment.Center
        ) {
            val poster = tmdbImage(record.posterUrl, "w185")
            if (poster != null) {
                AsyncImage(
                    model = poster,
                    contentDescription = stringResource(R.string.poster_description, record.title),
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize()
                )
            }
        }
        Column(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Text(
                text = record.title,
                style = MaterialTheme.typography.titleMedium,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis
            )
            val size = formatDownloadSize(
                status?.downloadedBytes?.takeIf { it > 0L } ?: record.sizeBytes
            )
            if (size.isNotEmpty()) {
                Text(
                    text = size,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            when (status?.phase) {
                DownloadPhase.Completed -> Text(
                    text = stringResource(R.string.downloaded),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.primary
                )
                DownloadPhase.Failed -> Text(
                    text = stringResource(R.string.download_failed),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error
                )
                DownloadPhase.Pending, DownloadPhase.Running -> {
                    val percent = status?.percent
                    Text(
                        text = if (status?.waitingForServer == true) {
                            stringResource(R.string.download_waiting)
                        } else if (percent == null) {
                            stringResource(R.string.download_in_progress)
                        } else {
                            stringResource(R.string.download_downloading, percent)
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    if (percent != null) {
                        LinearProgressIndicator(
                            progress = { percent / 100f },
                            modifier = Modifier.fillMaxWidth()
                        )
                    }
                }
                null -> Unit
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                when (status?.phase) {
                    DownloadPhase.Completed -> {
                        TextButton(onClick = onPlay) {
                            Text(stringResource(R.string.download_play))
                        }
                        TextButton(onClick = onDelete) {
                            Text(stringResource(R.string.download_delete))
                        }
                    }
                    DownloadPhase.Failed -> {
                        TextButton(onClick = onDelete) {
                            Text(stringResource(R.string.download_delete))
                        }
                    }
                    DownloadPhase.Pending, DownloadPhase.Running -> {
                        TextButton(onClick = onCancel) {
                            Text(stringResource(R.string.download_cancel_confirm))
                        }
                    }
                    null -> Unit
                }
            }
        }
    }
}
