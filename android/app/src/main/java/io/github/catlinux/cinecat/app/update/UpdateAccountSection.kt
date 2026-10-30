package io.github.catlinux.cinecat.app.update

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.focusable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import io.github.catlinux.cinecat.app.BuildConfig
import io.github.catlinux.cinecat.app.PrimaryActionButton
import io.github.catlinux.cinecat.app.R
import io.github.catlinux.cinecat.app.SecondaryActionButton
import kotlin.math.roundToInt
import kotlinx.coroutines.launch

@Composable
fun AccountUpdateSection(
    updater: AppUpdater,
    isTelevision: Boolean,
    initialFocusRequester: FocusRequester? = null
) {
    val state by updater.state.collectAsState()
    val autoEnabled by updater.autoEnabled.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var showPermission by remember { mutableStateOf(false) }
    var permissionFallback by remember { mutableStateOf(false) }
    val actionModifier = if (isTelevision) Modifier else Modifier.fillMaxWidth()

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = stringResource(R.string.update_section),
            style = MaterialTheme.typography.titleLarge,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = stringResource(R.string.version_format, BuildConfig.VERSION_NAME),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        AutoUpdateRow(
            enabled = autoEnabled,
            onChange = { enabled ->
                scope.launch { updater.setAutoEnabled(enabled) }
            }
        )
        SecondaryActionButton(
            isTelevision = isTelevision,
            label = stringResource(R.string.update_check),
            modifier = initialFocusRequester?.let { actionModifier.focusRequester(it) }
                ?: actionModifier,
            onClick = { updater.check(force = true) }
        )
        when (val current = state) {
            UpdateState.Idle -> Unit
            UpdateState.Checking -> Text(
                text = stringResource(R.string.update_checking),
                style = MaterialTheme.typography.bodyMedium
            )
            UpdateState.UpToDate -> Text(
                text = stringResource(R.string.update_up_to_date),
                style = MaterialTheme.typography.bodyMedium
            )
            UpdateState.ServerChecksDisabled -> Text(
                text = stringResource(R.string.update_server_disabled),
                style = MaterialTheme.typography.bodyMedium
            )
            UpdateState.NotPublished -> Text(
                text = stringResource(R.string.update_not_published),
                style = MaterialTheme.typography.bodyMedium
            )
            is UpdateState.Available -> {
                Text(
                    text = stringResource(
                        R.string.update_available,
                        current.update.version
                    ),
                    style = MaterialTheme.typography.bodyMedium
                )
                SecondaryActionButton(
                    isTelevision = isTelevision,
                    label = stringResource(R.string.update_download),
                    modifier = actionModifier,
                    onClick = { updater.download(current.update) }
                )
            }
            is UpdateState.Downloading -> {
                Text(
                    text = stringResource(
                        R.string.update_downloading,
                        (current.progress * 100).roundToInt()
                    ),
                    style = MaterialTheme.typography.bodyMedium
                )
                LinearProgressIndicator(
                    progress = { current.progress },
                    modifier = Modifier.fillMaxWidth()
                )
            }
            is UpdateState.Ready -> {
                Text(
                    text = stringResource(R.string.update_ready, current.update.version),
                    style = MaterialTheme.typography.bodyMedium
                )
                PrimaryActionButton(
                    isTelevision = isTelevision,
                    label = stringResource(R.string.update_install),
                    modifier = actionModifier,
                    onClick = {
                        if (!canInstallPackages(context)) {
                            showPermission = true
                        } else {
                            installDownloadedUpdate(
                                context = context,
                                apkFile = current.apkFile,
                                updater = updater
                            )
                        }
                    }
                )
            }
            is UpdateState.Failed -> {
                Text(
                    text = stringResource(
                        R.string.update_failed,
                        stringResource(failureMessage(current.reason))
                    ),
                    style = MaterialTheme.typography.bodyMedium
                )
                SecondaryActionButton(
                    isTelevision = isTelevision,
                    label = stringResource(R.string.update_retry),
                    modifier = actionModifier,
                    onClick = { updater.check(force = true) }
                )
            }
            UpdateState.Disabled -> Text(
                text = stringResource(R.string.update_debug_disabled),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }

    if (showPermission) {
        UpdateInstallPermissionDialog(
            fallback = permissionFallback,
            onOpenSettings = {
                if (openUnknownSourcesSettings(context)) {
                    showPermission = false
                } else {
                    permissionFallback = true
                }
            },
            onDismiss = { showPermission = false }
        )
    }
}

@Composable
private fun AutoUpdateRow(
    enabled: Boolean,
    onChange: (Boolean) -> Unit
) {
    var focused by remember { mutableStateOf(false) }
    fun toggle() = onChange(!enabled)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .background(
                if (focused) {
                    MaterialTheme.colorScheme.surfaceVariant
                } else {
                    androidx.compose.ui.graphics.Color.Transparent
                }
            )
            .clickable { toggle() }
            .focusable()
            .onFocusChanged { focused = it.isFocused }
            .onPreviewKeyEvent { event ->
                if (event.type == KeyEventType.KeyDown &&
                    event.key in setOf(
                        Key.DirectionCenter,
                        Key.Enter,
                        Key.NumPadEnter,
                        Key.Spacebar
                    )
                ) {
                    toggle()
                    true
                } else {
                    false
                }
            }
            .padding(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = stringResource(R.string.update_auto),
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier.weight(1f)
        )
        Switch(
            checked = enabled,
            onCheckedChange = null
        )
    }
}

private fun failureMessage(reason: UpdateFailure): Int = when (reason) {
    UpdateFailure.Network -> R.string.update_failure_network
    UpdateFailure.InvalidUrl -> R.string.update_failure_invalid_url
    UpdateFailure.Download -> R.string.update_failure_download
    UpdateFailure.TooLarge -> R.string.update_failure_too_large
    UpdateFailure.HashMismatch -> R.string.update_failure_hash
    UpdateFailure.InvalidApk -> R.string.update_failure_invalid_apk
    UpdateFailure.PackageMismatch -> R.string.update_failure_package
    UpdateFailure.VersionNotNewer -> R.string.update_failure_version
    UpdateFailure.SignatureMismatch -> R.string.update_failure_signature
    UpdateFailure.Install -> R.string.update_failure_install
    UpdateFailure.Disabled -> R.string.update_failure_disabled
}
