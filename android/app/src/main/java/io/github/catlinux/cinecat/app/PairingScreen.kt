package io.github.catlinux.cinecat.app

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import io.github.catlinux.cinecat.core.data.PairingState
import io.github.catlinux.cinecat.core.data.SessionRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.collect

@Composable
fun PairingScreen(
    repository: SessionRepository,
    isTelevision: Boolean,
    scope: CoroutineScope,
    onApproved: () -> Unit
) {
    val viewModel: PairingViewModel = viewModel {
        PairingViewModel(pairing = repository::startPairing, scope = scope)
    }
    LaunchedEffect(viewModel) {
        viewModel.approvedEvents.collect {
            onApproved()
        }
    }
    LaunchedEffect(Unit) {
        viewModel.restart()
    }
    DisposableEffect(Unit) {
        onDispose { viewModel.stop() }
    }
    val state by viewModel.state.collectAsState()

    val primaryFocus = remember { FocusRequester() }
    LaunchedEffect(state.pairing) {
        if (state.pairing is PairingState.Expired || state.pairing is PairingState.Failed) {
            runCatching { primaryFocus.requestFocus() }
        }
    }

    val horizontalPadding = if (isTelevision) 64.dp else 24.dp
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .windowInsetsPadding(WindowInsets.safeDrawing),
        contentAlignment = Alignment.Center
    ) {
        Column(
            modifier = Modifier
                .widthIn(max = if (isTelevision) 960.dp else 560.dp)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = horizontalPadding, vertical = if (isTelevision) 48.dp else 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(if (isTelevision) 18.dp else 12.dp)
        ) {
            Text(
                text = stringResource(R.string.app_name),
                style = MaterialTheme.typography.displaySmall,
                color = MaterialTheme.colorScheme.primary
            )
            Text(
                text = stringResource(R.string.pairing_title),
                style = if (isTelevision) {
                    MaterialTheme.typography.displayMedium
                } else {
                    MaterialTheme.typography.headlineLarge
                },
                textAlign = TextAlign.Center
            )
            Text(
                text = stringResource(R.string.pairing_lede),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center
            )

            when (val pairing = state.pairing) {
                null -> {
                    CircularProgressIndicator()
                    Text(stringResource(R.string.loading))
                }
                is PairingState.Waiting -> {
                    WaitingPairingContent(pairing, isTelevision)
                }
                is PairingState.Approved -> {
                    CircularProgressIndicator()
                    Text(stringResource(R.string.pairing_waiting))
                }
                PairingState.Expired -> {
                    Text(
                        text = stringResource(R.string.pairing_expired),
                        style = MaterialTheme.typography.titleLarge,
                        textAlign = TextAlign.Center
                    )
                    PrimaryActionButton(
                        isTelevision = isTelevision,
                        label = stringResource(R.string.pairing_new_code),
                        modifier = Modifier.focusRequester(primaryFocus),
                        onClick = viewModel::restart
                    )
                }
                is PairingState.Failed -> {
                    Text(
                        text = stringResource(R.string.pairing_failed),
                        style = MaterialTheme.typography.titleLarge,
                        color = MaterialTheme.colorScheme.error,
                        textAlign = TextAlign.Center
                    )
                    pairing.message?.takeIf { it.isNotBlank() }?.let { message ->
                        Text(
                            text = message,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            textAlign = TextAlign.Center
                        )
                    }
                    PrimaryActionButton(
                        isTelevision = isTelevision,
                        label = stringResource(R.string.pairing_retry),
                        modifier = Modifier.focusRequester(primaryFocus),
                        onClick = viewModel::restart
                    )
                }
            }
            Spacer(Modifier.height(if (isTelevision) 24.dp else 12.dp))
        }
    }
}

@Composable
private fun WaitingPairingContent(pairing: PairingState.Waiting, isTelevision: Boolean) {
    val qrImage = remember(pairing.verificationUrl) {
        qrCodeImage(pairing.verificationUrl, if (isTelevision) 900 else 640)
    }
    val remaining by produceState(
        initialValue = remainingSeconds(pairing.expiresAtMillis),
        pairing.expiresAtMillis
    ) {
        while (value > 0) {
            delay(1_000)
            value = remainingSeconds(pairing.expiresAtMillis)
        }
    }
    val verificationBase = remember(pairing.verificationUrl) {
        pairing.verificationUrl.substringBefore('?')
    }

    Text(
        text = stringResource(R.string.pairing_code_label),
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.onSurfaceVariant
    )
    Text(
        text = pairing.code,
        style = MaterialTheme.typography.displayLarge,
        color = MaterialTheme.colorScheme.primary,
        textAlign = TextAlign.Center
    )
    Image(
        bitmap = qrImage,
        contentDescription = stringResource(R.string.pairing_qr_description),
        modifier = Modifier
            .size(if (isTelevision) 240.dp else 200.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(12.dp))
    )
    Text(
        text = stringResource(R.string.pairing_open_url),
        style = MaterialTheme.typography.bodyLarge,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        textAlign = TextAlign.Center
    )
    Text(
        text = verificationBase,
        style = MaterialTheme.typography.titleMedium,
        color = MaterialTheme.colorScheme.primary,
        textAlign = TextAlign.Center
    )
    Text(
        text = stringResource(R.string.pairing_expires, remaining / 60, remaining % 60),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        textAlign = TextAlign.Center
    )
    Text(
        text = stringResource(R.string.pairing_waiting),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        textAlign = TextAlign.Center
    )
}

private fun remainingSeconds(expiresAtMillis: Long): Int =
    ((expiresAtMillis - System.currentTimeMillis()) / 1_000L)
        .coerceAtLeast(0L)
        .toInt()
