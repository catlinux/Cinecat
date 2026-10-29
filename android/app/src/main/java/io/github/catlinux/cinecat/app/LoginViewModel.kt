package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.data.LoginResult
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class LoginError {
    InvalidCredentials,
    TooManyAttempts,
    Network
}

data class LoginUiState(
    val username: String = "",
    val password: String = "",
    val passwordVisible: Boolean = false,
    val submitting: Boolean = false,
    val error: LoginError? = null,
    val retrySeconds: Int = 0,
    val signedIn: Boolean = false
) {
    val canSubmit: Boolean
        get() = !submitting &&
            retrySeconds == 0 &&
            username.isNotBlank() &&
            password.isNotEmpty()
}

class LoginViewModel(
    private val login: suspend (String, String) -> LoginResult,
    private val scope: CoroutineScope
) : ViewModel() {
    private val _state = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = _state.asStateFlow()

    private val _signedInEvents = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val signedInEvents: SharedFlow<Unit> = _signedInEvents.asSharedFlow()

    private var countdownJob: Job? = null
    private var loginJob: Job? = null

    fun reset() {
        loginJob?.cancel()
        loginJob = null
        countdownJob?.cancel()
        countdownJob = null
        _state.value = LoginUiState()
    }

    fun onUsernameChanged(value: String) {
        _state.update { it.copy(username = value, error = null) }
    }

    fun onPasswordChanged(value: String) {
        _state.update { it.copy(password = value, error = null) }
    }

    fun togglePasswordVisibility() {
        _state.update { it.copy(passwordVisible = !it.passwordVisible) }
    }

    fun login() {
        val current = _state.value
        if (!current.canSubmit) {
            return
        }
        _state.update { it.copy(submitting = true, error = null) }
        loginJob = scope.launch {
            val result = try {
                login(current.username, current.password)
            } catch (error: CancellationException) {
                throw error
            } catch (_: Exception) {
                LoginResult.Failure(null)
            }
            when (result) {
                LoginResult.Success -> {
                    _state.update { it.copy(submitting = false, signedIn = true) }
                    _signedInEvents.emit(Unit)
                }
                LoginResult.InvalidCredentials -> {
                    _state.update {
                        it.copy(
                            submitting = false,
                            error = LoginError.InvalidCredentials
                        )
                    }
                }
                is LoginResult.RateLimited -> {
                    startCountdown(result.retryAfterSeconds)
                }
                is LoginResult.Failure -> {
                    _state.update {
                        it.copy(submitting = false, error = LoginError.Network)
                    }
                }
            }
        }
    }

    private fun startCountdown(retryAfterSeconds: Long?) {
        countdownJob?.cancel()
        val seconds = (retryAfterSeconds ?: 1L)
            .coerceIn(1L, Int.MAX_VALUE.toLong())
            .toInt()
        _state.update {
            it.copy(
                submitting = false,
                error = LoginError.TooManyAttempts,
                retrySeconds = seconds
            )
        }
        countdownJob = scope.launch {
            var remaining = seconds
            while (remaining > 0) {
                delay(1_000)
                remaining--
                _state.update {
                    it.copy(
                        retrySeconds = remaining,
                        error = if (remaining > 0) LoginError.TooManyAttempts else null
                    )
                }
            }
        }
    }

    override fun onCleared() {
        loginJob?.cancel()
        countdownJob?.cancel()
    }
}
