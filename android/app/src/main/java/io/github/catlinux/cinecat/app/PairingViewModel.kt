package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.data.PairingState
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.launch

data class PairingUiState(
    val pairing: PairingState? = null,
    val approved: Boolean = false
)

class PairingViewModel(
    private val pairing: () -> Flow<PairingState>,
    private val scope: CoroutineScope
) : ViewModel() {
    private val _state = MutableStateFlow(PairingUiState())
    val state: StateFlow<PairingUiState> = _state.asStateFlow()

    private val _approvedEvents = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val approvedEvents: SharedFlow<Unit> = _approvedEvents.asSharedFlow()

    private var job: Job? = null

    fun start() {
        if (job?.isActive == true) {
            return
        }
        job = scope.launch {
            pairing()
                .catch { error ->
                    _state.value = PairingUiState(PairingState.Failed(error.message))
                }
                .collect { pairingState ->
                    _state.value = PairingUiState(
                        pairing = pairingState,
                        approved = pairingState is PairingState.Approved
                    )
                    if (pairingState is PairingState.Approved) {
                        _approvedEvents.emit(Unit)
                    }
                }
        }
    }

    fun restart() {
        job?.cancel()
        job = null
        _state.value = PairingUiState()
        start()
    }

    fun stop() {
        job?.cancel()
        job = null
    }

    override fun onCleared() {
        stop()
    }
}
