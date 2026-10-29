package io.github.catlinux.cinecat.app

import android.content.res.Configuration
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.LocalIndication
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import io.github.catlinux.cinecat.app.update.UpdateReadyDialog
import io.github.catlinux.cinecat.app.update.UpdateInstallPermissionDialog
import io.github.catlinux.cinecat.app.update.UpdateState
import io.github.catlinux.cinecat.app.update.canInstallPackages
import io.github.catlinux.cinecat.app.update.installDownloadedUpdate
import io.github.catlinux.cinecat.app.update.openUnknownSourcesSettings
import io.github.catlinux.cinecat.ui.common.theme.CinecatTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val isTv = isTelevision(resources.configuration.uiMode and Configuration.UI_MODE_TYPE_MASK)
        val container = (application as CinecatApplication).container
        setContent {
            CinecatTheme {
                CompositionLocalProvider(LocalIsTelevision provides isTv) {
                    if (isTv) {
                        val color = MaterialTheme.colorScheme.tertiary
                        val indication = remember(color) { FocusIndication(color) }
                        CompositionLocalProvider(LocalIndication provides indication) {
                            CinecatApp(container = container, isTelevision = isTv)
                            UpdateCoordinator(container)
                        }
                    } else {
                        CinecatApp(container = container, isTelevision = isTv)
                        UpdateCoordinator(container)
                    }
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        val container = (application as CinecatApplication).container
        container.appUpdater.check()
    }
}

@Composable
private fun UpdateCoordinator(container: AppContainer) {
    val state by container.appUpdater.state.collectAsState()
    val autoEnabled by container.appUpdater.autoEnabled.collectAsState()
    val context = LocalContext.current
    var dismissedVersion by rememberSaveable { mutableStateOf<String?>(null) }
    var showPermission by remember { mutableStateOf(false) }
    var permissionFallback by remember { mutableStateOf(false) }

    LaunchedEffect(autoEnabled, state) {
        val current = state
        if (autoEnabled && current is UpdateState.Available) {
            container.appUpdater.download(current.update)
        }
    }

    val ready = state as? UpdateState.Ready
    if (ready != null && ready.update.version != dismissedVersion && !showPermission) {
        UpdateReadyDialog(
            update = ready.update,
            onInstall = {
                if (!canInstallPackages(context)) {
                    showPermission = true
                } else {
                    installDownloadedUpdate(
                        context = context,
                        apkFile = ready.apkFile,
                        updater = container.appUpdater
                    )
                    dismissedVersion = ready.update.version
                }
            },
            onLater = {
                dismissedVersion = ready.update.version
            }
        )
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
