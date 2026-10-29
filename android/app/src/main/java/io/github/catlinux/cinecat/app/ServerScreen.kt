package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import io.github.catlinux.cinecat.core.data.SessionRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.collect

@Composable
fun ServerScreen(
    repository: SessionRepository,
    isTelevision: Boolean,
    scope: CoroutineScope,
    onSaved: () -> Unit
) {
    val initialServer = remember { repository.currentServer()?.toString().orEmpty() }
    val initialLocalAddress = remember { repository.currentLocalAddress()?.hostAddress.orEmpty() }
    val viewModel: ServerViewModel = viewModel {
        ServerViewModel(
            initialServer = initialServer,
            initialLocalAddress = initialLocalAddress,
            checkServer = { address -> repository.checkServer(address) },
            saveServer = { address, localAddress -> repository.setServer(address, localAddress) },
            scope = scope
        )
    }
    LaunchedEffect(Unit) {
        viewModel.reset(initialServer, initialLocalAddress)
    }
    val state by viewModel.state.collectAsState()
    LaunchedEffect(viewModel) {
        viewModel.savedEvents.collect {
            onSaved()
        }
    }

    val horizontalPadding = if (isTelevision) 64.dp else 24.dp
    Scaffold(
        modifier = Modifier.fillMaxSize(),
        contentWindowInsets = WindowInsets.safeDrawing
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .imePadding(),
            contentAlignment = Alignment.TopCenter
        ) {
            Column(
                modifier = Modifier
                    .widthIn(max = if (isTelevision) 960.dp else 560.dp)
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = horizontalPadding, vertical = if (isTelevision) 56.dp else 28.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Text(
                    text = stringResource(R.string.app_name),
                    style = MaterialTheme.typography.displaySmall,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = stringResource(R.string.server_title),
                    style = if (isTelevision) {
                        MaterialTheme.typography.displayMedium
                    } else {
                        MaterialTheme.typography.headlineLarge
                    }
                )
                Text(
                    text = stringResource(R.string.server_lede),
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                OutlinedTextField(
                    value = state.server,
                    onValueChange = viewModel::onServerChanged,
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(stringResource(R.string.server_address_label)) },
                    placeholder = { Text(stringResource(R.string.server_address_hint)) },
                    singleLine = true,
                    enabled = !state.checking,
                    isError = state.fieldError == ServerFieldError.Address,
                    supportingText = {
                        if (state.fieldError == ServerFieldError.Address) {
                            Text(stringResource(R.string.server_error_address))
                        }
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Uri,
                        imeAction = ImeAction.Next
                    )
                )
                OutlinedTextField(
                    value = state.localAddress,
                    onValueChange = viewModel::onLocalAddressChanged,
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(stringResource(R.string.server_local_label)) },
                    singleLine = true,
                    enabled = !state.checking,
                    isError = state.fieldError == ServerFieldError.LocalAddress,
                    supportingText = {
                        if (state.fieldError == ServerFieldError.LocalAddress) {
                            Text(stringResource(R.string.server_error_local))
                        } else {
                            Text(stringResource(R.string.server_local_help))
                        }
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Text,
                        imeAction = ImeAction.Done
                    )
                )
                PrimaryActionButton(
                    isTelevision = isTelevision,
                    label = stringResource(
                        if (state.checking) R.string.server_checking else R.string.server_check
                    ),
                    enabled = !state.checking,
                    onClick = viewModel::check
                )
                state.check?.let { check ->
                    Text(
                        text = check.message(),
                        style = MaterialTheme.typography.bodyLarge,
                        color = if (check is ServerCheckUi.Compatible) {
                            MaterialTheme.colorScheme.primary
                        } else {
                            MaterialTheme.colorScheme.error
                        }
                    )
                    if (check is ServerCheckUi.Compatible) {
                        PrimaryActionButton(
                            isTelevision = isTelevision,
                            label = stringResource(R.string.server_continue),
                            onClick = viewModel::save
                        )
                    }
                }
                if (state.saveFailed) {
                    Text(
                        text = stringResource(R.string.server_save_error),
                        color = MaterialTheme.colorScheme.error,
                        style = MaterialTheme.typography.bodyLarge
                    )
                }
                Spacer(Modifier.height(8.dp))
            }
        }
    }
}

@Composable
private fun ServerCheckUi.message(): String = when (this) {
    is ServerCheckUi.Compatible -> stringResource(
        R.string.server_compatible,
        version.server
    )
    ServerCheckUi.ServerTooOld -> stringResource(R.string.server_too_old)
    ServerCheckUi.ApiIncompatible -> stringResource(R.string.server_api_incompatible)
    is ServerCheckUi.AppTooOld -> stringResource(R.string.server_app_too_old, minClient)
    ServerCheckUi.NotCinecat -> stringResource(R.string.server_not_cinecat)
    ServerCheckUi.Unreachable -> stringResource(R.string.server_unreachable)
}
