package io.github.catlinux.cinecat.app

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.core.content.ContextCompat
import io.github.catlinux.cinecat.core.api.DownloadRecord
import io.github.catlinux.cinecat.core.api.MediaItem
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
    var preparing by remember { mutableStateOf(false) }
    var cancelRecord by remember { mutableStateOf<DownloadRecord?>(null) }
    var actionRecord by remember { mutableStateOf<DownloadRecord?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val statuses by downloads.statuses.collectAsState()
    val status = record?.let { statuses[it.itemId] }

    fun startDownload() {
        preparing = true
        downloads.startAsync(item, caps) { result ->
            when (result) {
                is DownloadStartResult.Failed -> errorMessage = result.message
                is DownloadStartResult.Started -> Unit
            }
            preparing = false
        }
    }

    // Sin el permiso la descarga sigue igualmente; solo falta la notificación.
    val notificationPermission = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { startDownload() }

    val phase = status?.phase
    val label = when (downloadLabel(record != null, preparing, phase, status?.waitingForServer == true)) {
        DownloadLabel.Preparing -> stringResource(R.string.download_preparing)
        DownloadLabel.Waiting -> stringResource(R.string.download_waiting)
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
                val needsPermission = Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
                    ContextCompat.checkSelfPermission(
                        context,
                        Manifest.permission.POST_NOTIFICATIONS
                    ) != PackageManager.PERMISSION_GRANTED
                if (needsPermission) {
                    notificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
                } else {
                    startDownload()
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
