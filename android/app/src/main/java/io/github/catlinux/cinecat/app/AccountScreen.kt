package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import io.github.catlinux.cinecat.app.update.AccountUpdateSection
import io.github.catlinux.cinecat.app.update.AppUpdater
import io.github.catlinux.cinecat.core.data.SessionRepository
import io.github.catlinux.cinecat.core.data.SessionState
import kotlinx.coroutines.launch

@Composable
fun AccountScreen(
    repository: SessionRepository,
    session: SessionState.SignedIn,
    updater: AppUpdater,
    isTelevision: Boolean,
    onBack: () -> Unit,
    onChangeServer: () -> Unit
) {
    val scope = rememberCoroutineScope()
    val server = session.server.toString().removeSuffix("/")
    val updateFocusRequester = remember { FocusRequester() }

    LaunchedEffect(isTelevision) {
        if (isTelevision) {
            withFrameNanos { }
            runCatching { updateFocusRequester.requestFocus() }
        }
    }

    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.TopCenter
    ) {
        if (isTelevision) {
            TextButton(
                onClick = onBack,
                modifier = Modifier
                    .align(Alignment.TopStart)
                    .padding(16.dp)
            ) {
                Text(stringResource(R.string.back))
            }
            Row(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(start = 64.dp, end = 64.dp, top = 64.dp, bottom = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(48.dp),
                verticalAlignment = Alignment.Top
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Text(
                        text = stringResource(R.string.account_title),
                        style = MaterialTheme.typography.headlineMedium,
                        color = MaterialTheme.colorScheme.primary
                    )
                    Text(
                        text = stringResource(R.string.home_signed_in_as, session.user),
                        style = MaterialTheme.typography.titleMedium
                    )
                    Text(
                        text = stringResource(R.string.home_server, server),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(Modifier.height(8.dp))
                    SecondaryActionButton(
                        isTelevision = true,
                        fillWidth = false,
                        label = stringResource(R.string.home_logout),
                        onClick = { scope.launch { repository.logout() } }
                    )
                    SecondaryActionButton(
                        isTelevision = true,
                        fillWidth = false,
                        label = stringResource(R.string.home_change_server),
                        onClick = onChangeServer
                    )
                }
                Column(modifier = Modifier.weight(1f)) {
                    AccountUpdateSection(
                        updater = updater,
                        isTelevision = true,
                        initialFocusRequester = updateFocusRequester
                    )
                }
            }
        } else {
            Column(
                modifier = Modifier
                    .widthIn(max = 560.dp)
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 24.dp, vertical = 16.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    TextButton(onClick = onBack) {
                        Text(stringResource(R.string.back))
                    }
                    Spacer(Modifier.weight(1f))
                    Text(
                        text = stringResource(R.string.account_title),
                        style = MaterialTheme.typography.titleLarge
                    )
                }
                Text(
                    text = stringResource(R.string.app_name),
                    style = MaterialTheme.typography.displayLarge,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = stringResource(R.string.home_signed_in_as, session.user),
                    style = MaterialTheme.typography.titleLarge,
                    textAlign = TextAlign.Center
                )
                Text(
                    text = stringResource(R.string.home_server, server),
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center
                )
                Spacer(Modifier.height(8.dp))
                AccountUpdateSection(updater = updater, isTelevision = false)
                Spacer(Modifier.height(8.dp))
                PrimaryActionButton(
                    isTelevision = false,
                    label = stringResource(R.string.home_logout),
                    modifier = Modifier.fillMaxWidth(),
                    onClick = { scope.launch { repository.logout() } }
                )
                SecondaryActionButton(
                    isTelevision = false,
                    label = stringResource(R.string.home_change_server),
                    modifier = Modifier.fillMaxWidth(),
                    onClick = onChangeServer
                )
            }
        }
    }
}
