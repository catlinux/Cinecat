package io.github.catlinux.cinecat.app.update

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.provider.Settings
import androidx.core.net.toUri
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import io.github.catlinux.cinecat.app.R
import io.github.catlinux.cinecat.core.api.ClientUpdate
import java.io.File
import kotlinx.coroutines.delay

fun canInstallPackages(context: Context): Boolean =
    context.packageManager.canRequestPackageInstalls()

fun openUnknownSourcesSettings(context: Context): Boolean {
    return try {
        context.startActivity(
            Intent(
                Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                "package:${context.packageName}".toUri()
            ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        )
        true
    } catch (_: ActivityNotFoundException) {
        false
    } catch (_: Exception) {
        false
    }
}

fun installDownloadedUpdate(context: Context, apkFile: File, updater: AppUpdater) {
    updater.install(context.applicationContext, apkFile)
}

@Composable
fun UpdateReadyDialog(
    update: ClientUpdate,
    onInstall: () -> Unit,
    onLater: () -> Unit
) {
    val installFocus = remember { FocusRequester() }
    LaunchedEffect(installFocus) {
        delay(100)
        runCatching { installFocus.requestFocus() }
    }
    AlertDialog(
        onDismissRequest = onLater,
        title = { Text(stringResource(R.string.update_ready_title)) },
        text = { Text(stringResource(R.string.update_ready, update.version)) },
        confirmButton = {
            Button(
                onClick = onInstall,
                modifier = Modifier.focusRequester(installFocus)
            ) {
                Text(stringResource(R.string.update_install))
            }
        },
        dismissButton = {
            TextButton(onClick = onLater) {
                Text(stringResource(R.string.update_later))
            }
        }
    )
}

@Composable
fun UpdateInstallPermissionDialog(
    fallback: Boolean,
    onOpenSettings: () -> Unit,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.update_permission_title)) },
        text = {
            Text(
                stringResource(
                    if (fallback) {
                        R.string.update_permission_fallback
                    } else {
                        R.string.update_permission_message
                    }
                )
            )
        },
        confirmButton = {
            if (fallback) {
                TextButton(onClick = onDismiss) {
                    Text(stringResource(R.string.update_close))
                }
            } else {
                TextButton(onClick = onOpenSettings) {
                    Text(stringResource(R.string.update_open_settings))
                }
            }
        },
        dismissButton = if (fallback) {
            null
        } else {
            {
                TextButton(onClick = onDismiss) {
                    Text(stringResource(R.string.update_close))
                }
            }
        }
    )
}

@Composable
fun UpdateInstallingOverlay(version: String) {
    Dialog(
        onDismissRequest = {},
        properties = DialogProperties(
            dismissOnBackPress = false,
            dismissOnClickOutside = false,
            usePlatformDefaultWidth = false,
            decorFitsSystemWindows = false
        )
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(MaterialTheme.colorScheme.background),
            contentAlignment = Alignment.Center
        ) {
            Column(
                modifier = Modifier.padding(32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(18.dp)
            ) {
                CircularProgressIndicator(color = MaterialTheme.colorScheme.tertiary)
                Text(
                    text = stringResource(R.string.update_installing_title),
                    style = MaterialTheme.typography.headlineMedium,
                    color = MaterialTheme.colorScheme.primary,
                    textAlign = TextAlign.Center
                )
                Text(
                    text = stringResource(R.string.update_installing_version, version),
                    style = MaterialTheme.typography.titleLarge,
                    color = MaterialTheme.colorScheme.onBackground,
                    textAlign = TextAlign.Center
                )
                Text(
                    text = stringResource(R.string.update_installing_note),
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center
                )
            }
        }
    }
}

@Composable
fun UpdateCompletedDialog(version: String, onDismiss: () -> Unit) {
    val acceptFocus = remember { FocusRequester() }
    LaunchedEffect(acceptFocus) {
        delay(100)
        runCatching { acceptFocus.requestFocus() }
    }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.update_completed_title)) },
        text = { Text(stringResource(R.string.update_completed_message, version)) },
        confirmButton = {
            Button(
                onClick = onDismiss,
                modifier = Modifier.focusRequester(acceptFocus)
            ) {
                Text(stringResource(R.string.update_accept))
            }
        }
    )
}
