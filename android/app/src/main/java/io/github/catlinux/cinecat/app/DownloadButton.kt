package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import io.github.catlinux.cinecat.core.api.DownloadRecord
import io.github.catlinux.cinecat.core.api.MediaItem
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun DownloadButton(
    item: MediaItem,
    downloads: DownloadRepository,
    caps: String,
    onPlayLocal: (DownloadRecord) -> Unit,
    compact: Boolean = false
) {
    if (LocalIsTelevision.current) {
        return
    }
    val allRecords by downloads.records.collectAsState()
    val record = allRecords.downloads.firstOrNull { it.itemId == item.id }
    var status by remember(record?.downloadId) { mutableStateOf<DownloadStatus?>(null) }
    var preparing by remember { mutableStateOf(false) }
    var cancelRecord by remember { mutableStateOf<DownloadRecord?>(null) }
    var actionRecord by remember { mutableStateOf<DownloadRecord?>(null) }
    var blockedKind by remember { mutableStateOf<DownloadBlockedKind?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(record?.downloadId) {
        val current = record
        if (current == null) {
            status = null
            return@LaunchedEffect
        }
        while (true) {
            val next = downloads.status(current)
            status = next
            if (next.phase == DownloadPhase.Completed || next.phase == DownloadPhase.Failed) {
                break
            }
            delay(1000)
        }
    }

    val phase = status?.phase
    val label = when (downloadLabel(record != null, preparing, phase)) {
        DownloadLabel.Preparing -> stringResource(R.string.download_preparing)
        DownloadLabel.Download -> stringResource(R.string.download_button)
        DownloadLabel.Downloaded -> stringResource(R.string.downloaded)
        DownloadLabel.Failed -> stringResource(R.string.download_failed)
        DownloadLabel.Downloading -> {
            val percent = status?.percent
            if (percent == null) {
                stringResource(R.string.download_in_progress)
            } else {
                stringResource(R.string.download_downloading, percent)
            }
        }
    }

    val onClick: () -> Unit = {
        when {
            preparing -> Unit
            record == null -> {
                preparing = true
                scope.launch {
                    when (val result = downloads.start(item, caps)) {
                        is DownloadStartResult.NotDownloadable ->
                            blockedKind = downloadBlockedKind(result.reason)
                        is DownloadStartResult.Failed -> errorMessage = result.message
                        is DownloadStartResult.Started -> Unit
                    }
                    preparing = false
                }
            }
            phase == DownloadPhase.Completed || phase == DownloadPhase.Failed -> {
                actionRecord = record
            }
            else -> cancelRecord = record
        }
    }

    if (compact) {
        TextButton(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
            Text(label)
        }
    } else {
        OutlinedButton(onClick = onClick, modifier = Modifier.fillMaxWidth()) {
            Text(label)
        }
    }

    cancelRecord?.let { pending ->
        AlertDialog(
            onDismissRequest = { cancelRecord = null },
            title = { Text(stringResource(R.string.download_cancel_title)) },
            text = {
                Text(stringResource(R.string.download_cancel_message, pending.title))
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        cancelRecord = null
                        scope.launch { downloads.cancel(pending) }
                    }
                ) {
                    Text(stringResource(R.string.download_cancel_confirm))
                }
            },
            dismissButton = {
                TextButton(onClick = { cancelRecord = null }) {
                    Text(stringResource(R.string.download_close))
                }
            }
        )
    }

    actionRecord?.let { downloaded ->
        val failed = status?.phase == DownloadPhase.Failed
        AlertDialog(
            onDismissRequest = { actionRecord = null },
            title = {
                Text(
                    stringResource(
                        if (failed) R.string.download_delete_title else R.string.downloaded
                    )
                )
            },
            text = {
                Text(
                    if (failed) {
                        stringResource(R.string.download_delete_message, downloaded.title)
                    } else {
                        downloaded.title
                    }
                )
            },
            confirmButton = {
                Row {
                    if (!failed) {
                        TextButton(
                            onClick = {
                                actionRecord = null
                                onPlayLocal(downloaded)
                            }
                        ) {
                            Text(stringResource(R.string.download_play))
                        }
                    }
                    TextButton(
                        onClick = {
                            actionRecord = null
                            scope.launch { downloads.delete(downloaded) }
                        }
                    ) {
                        Text(stringResource(R.string.download_delete))
                    }
                }
            },
            dismissButton = {
                TextButton(onClick = { actionRecord = null }) {
                    Text(stringResource(R.string.download_close))
                }
            }
        )
    }

    blockedKind?.let { kind ->
        AlertDialog(
            onDismissRequest = { blockedKind = null },
            title = { Text(stringResource(R.string.download_dialog_title)) },
            text = {
                Text(
                    stringResource(
                        when (kind) {
                            DownloadBlockedKind.Audio -> R.string.download_blocked_audio
                            DownloadBlockedKind.Video -> R.string.download_blocked_video
                            DownloadBlockedKind.Container -> R.string.download_blocked_container
                            DownloadBlockedKind.Other -> R.string.download_blocked_other
                        }
                    )
                )
            },
            confirmButton = {
                TextButton(onClick = { blockedKind = null }) {
                    Text(stringResource(R.string.download_close))
                }
            }
        )
    }

    errorMessage?.let { message ->
        AlertDialog(
            onDismissRequest = { errorMessage = null },
            title = { Text(stringResource(R.string.download_dialog_title)) },
            text = {
                Text(
                    message.takeIf { it.isNotBlank() }
                        ?: stringResource(R.string.download_error)
                )
            },
            confirmButton = {
                TextButton(onClick = { errorMessage = null }) {
                    Text(stringResource(R.string.download_close))
                }
            }
        )
    }
}
