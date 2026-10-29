package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ServerVersion
import io.github.catlinux.cinecat.core.data.Compatibility
import io.github.catlinux.cinecat.core.data.LocalAddress
import io.github.catlinux.cinecat.core.data.ServerAddress
import io.github.catlinux.cinecat.core.data.ServerCheckResult
import java.net.InetAddress
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import okhttp3.HttpUrl

enum class ServerFieldError {
    Address,
    LocalAddress
}

sealed interface ServerCheckUi {
    data class Compatible(val version: ServerVersion) : ServerCheckUi
    data object ServerTooOld : ServerCheckUi
    data object ApiIncompatible : ServerCheckUi
    data class AppTooOld(val minClient: String) : ServerCheckUi
    data object NotCinecat : ServerCheckUi
    data object Unreachable : ServerCheckUi
}

data class ServerReady(
    val origin: HttpUrl,
    val localAddress: InetAddress?
)

data class ServerUiState(
    val server: String = "",
    val localAddress: String = "",
    val checking: Boolean = false,
    val fieldError: ServerFieldError? = null,
    val check: ServerCheckUi? = null,
    val ready: ServerReady? = null,
    val saved: Boolean = false,
    val saveFailed: Boolean = false
)

class ServerViewModel(
    initialServer: String,
    initialLocalAddress: String,
    private val checkServer: suspend (HttpUrl) -> ServerCheckResult,
    private val saveServer: suspend (HttpUrl, InetAddress?) -> Unit,
    private val scope: CoroutineScope
) : ViewModel() {
    private val _state = MutableStateFlow(
        ServerUiState(
            server = initialServer.removeSuffix("/"),
            localAddress = initialLocalAddress
        )
    )
    val state: StateFlow<ServerUiState> = _state.asStateFlow()

    private val _savedEvents = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val savedEvents: SharedFlow<Unit> = _savedEvents.asSharedFlow()

    private var workJob: Job? = null

    fun reset(initialServer: String, initialLocalAddress: String) {
        workJob?.cancel()
        workJob = null
        _state.value = ServerUiState(
            server = initialServer.removeSuffix("/"),
            localAddress = initialLocalAddress
        )
    }

    fun onServerChanged(value: String) {
        _state.update {
            it.copy(
                // Algunos teclados añaden un espacio tras cada punto; en una dirección nunca es válido.
                server = value.filterNot(Char::isWhitespace),
                fieldError = null,
                check = null,
                ready = null,
                saveFailed = false
            )
        }
    }

    fun onLocalAddressChanged(value: String) {
        _state.update {
            it.copy(
                localAddress = value.filterNot(Char::isWhitespace),
                fieldError = null,
                check = null,
                ready = null,
                saveFailed = false
            )
        }
    }

    fun check() {
        val current = _state.value
        if (current.checking) {
            return
        }
        val origin = ServerAddress.parse(current.server).getOrElse {
            _state.update { it.copy(fieldError = ServerFieldError.Address, check = null, ready = null) }
            return
        }
        val localAddress = LocalAddress.parse(current.localAddress).getOrElse {
            _state.update {
                it.copy(fieldError = ServerFieldError.LocalAddress, check = null, ready = null)
            }
            return
        }
        _state.update {
            it.copy(
                checking = true,
                fieldError = null,
                check = null,
                ready = null,
                saveFailed = false
            )
        }
        workJob = scope.launch {
            val result = try {
                checkServer(origin)
            } catch (error: CancellationException) {
                throw error
            } catch (_: Exception) {
                ServerCheckResult.Unreachable
            }
            val check = result.toUi()
            _state.update {
                it.copy(
                    checking = false,
                    check = check,
                    ready = if (check is ServerCheckUi.Compatible) {
                        ServerReady(origin, localAddress)
                    } else {
                        null
                    }
                )
            }
        }
    }

    fun save() {
        val ready = _state.value.ready ?: return
        if (_state.value.saved) {
            return
        }
        workJob = scope.launch {
            val saved = try {
                saveServer(ready.origin, ready.localAddress)
                true
            } catch (error: CancellationException) {
                throw error
            } catch (_: Exception) {
                false
            }
            _state.update { it.copy(saved = saved, saveFailed = !saved) }
            if (saved) {
                _savedEvents.emit(Unit)
            }
        }
    }

    override fun onCleared() {
        workJob?.cancel()
    }
}

private fun ServerCheckResult.toUi(): ServerCheckUi = when (this) {
    is ServerCheckResult.Compatible -> ServerCheckUi.Compatible(version)
    is ServerCheckResult.Incompatible -> when (val compatibility = reason) {
        Compatibility.ServerTooOld -> ServerCheckUi.ServerTooOld
        Compatibility.ApiIncompatible -> ServerCheckUi.ApiIncompatible
        is Compatibility.AppTooOld -> ServerCheckUi.AppTooOld(compatibility.minClient)
        Compatibility.Compatible -> ServerCheckUi.Compatible(version)
    }
    ServerCheckResult.NotCinecat -> ServerCheckUi.NotCinecat
    ServerCheckResult.Unreachable -> ServerCheckUi.Unreachable
}
