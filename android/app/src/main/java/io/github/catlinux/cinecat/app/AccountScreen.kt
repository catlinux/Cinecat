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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import io.github.catlinux.cinecat.core.data.SessionRepository
import io.github.catlinux.cinecat.core.data.SessionState
import kotlinx.coroutines.launch

@Composable
fun AccountScreen(
    repository: SessionRepository,
    session: SessionState.SignedIn,
    isTelevision: Boolean,
    onBack: () -> Unit,
    onChangeServer: () -> Unit
) {
    val scope = rememberCoroutineScope()
    val server = session.server.toString().removeSuffix("/")
    val horizontalPadding = if (isTelevision) 64.dp else 24.dp
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.TopCenter
    ) {
        Column(
            modifier = Modifier
                .widthIn(max = if (isTelevision) 900.dp else 560.dp)
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = horizontalPadding, vertical = 16.dp),
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
            Text(
                text = stringResource(R.string.version_format, BuildConfig.VERSION_NAME),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(Modifier.height(8.dp))
            PrimaryActionButton(
                isTelevision = isTelevision,
                label = stringResource(R.string.home_logout),
                modifier = Modifier.fillMaxWidth(),
                onClick = { scope.launch { repository.logout() } }
            )
            SecondaryActionButton(
                isTelevision = isTelevision,
                label = stringResource(R.string.home_change_server),
                modifier = Modifier.fillMaxWidth(),
                onClick = onChangeServer
            )
        }
    }
}
