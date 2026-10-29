package io.github.catlinux.cinecat.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import io.github.catlinux.cinecat.core.data.SessionRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.collect

@Composable
fun LoginScreen(
    repository: SessionRepository,
    scope: CoroutineScope,
    onUseCode: () -> Unit,
    onSignedIn: () -> Unit
) {
    val viewModel: LoginViewModel = viewModel {
        LoginViewModel(login = repository::login, scope = scope)
    }
    LaunchedEffect(Unit) {
        viewModel.reset()
    }
    val state by viewModel.state.collectAsState()
    LaunchedEffect(viewModel) {
        viewModel.signedInEvents.collect {
            onSignedIn()
        }
    }

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
                    .widthIn(max = 520.dp)
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 24.dp, vertical = 32.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Text(
                    text = stringResource(R.string.app_name),
                    style = MaterialTheme.typography.displaySmall,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = stringResource(R.string.login_title),
                    style = MaterialTheme.typography.headlineLarge
                )
                Text(
                    text = stringResource(R.string.login_lede),
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                OutlinedTextField(
                    value = state.username,
                    onValueChange = viewModel::onUsernameChanged,
                    modifier = Modifier.fillMaxWidth().dpadFieldNavigation(),
                    label = { Text(stringResource(R.string.login_username)) },
                    singleLine = true,
                    enabled = !state.submitting && state.retrySeconds == 0,
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Text,
                        imeAction = ImeAction.Next
                    )
                )
                OutlinedTextField(
                    value = state.password,
                    onValueChange = viewModel::onPasswordChanged,
                    modifier = Modifier.fillMaxWidth().dpadFieldNavigation(),
                    label = { Text(stringResource(R.string.login_password)) },
                    singleLine = true,
                    enabled = !state.submitting && state.retrySeconds == 0,
                    visualTransformation = if (state.passwordVisible) {
                        VisualTransformation.None
                    } else {
                        PasswordVisualTransformation()
                    },
                    trailingIcon = {
                        TextButton(onClick = viewModel::togglePasswordVisibility) {
                            Text(
                                stringResource(
                                    if (state.passwordVisible) {
                                        R.string.login_hide_password
                                    } else {
                                        R.string.login_show_password
                                    }
                                )
                            )
                        }
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Done
                    ),
                    keyboardActions = KeyboardActions(onDone = { viewModel.login() })
                )
                state.error?.let { error ->
                    Text(
                        text = error.message(state.retrySeconds),
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.error
                    )
                }
                Button(
                    onClick = viewModel::login,
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 52.dp),
                    enabled = state.canSubmit
                ) {
                    if (state.submitting) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(22.dp),
                            strokeWidth = 2.dp,
                            color = MaterialTheme.colorScheme.onPrimary
                        )
                    } else {
                        Text(
                            text = stringResource(R.string.login_submit),
                            style = MaterialTheme.typography.titleMedium
                        )
                    }
                }
                TextButton(
                    onClick = onUseCode,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !state.submitting
                ) {
                    Text(stringResource(R.string.login_use_code))
                }
                Spacer(Modifier.height(8.dp))
            }
        }
    }
}

@Composable
private fun LoginError.message(retrySeconds: Int): String = when (this) {
    LoginError.InvalidCredentials -> stringResource(R.string.login_invalid_credentials)
    LoginError.TooManyAttempts -> stringResource(R.string.login_too_many, retrySeconds)
    LoginError.Network -> stringResource(R.string.login_network)
}
