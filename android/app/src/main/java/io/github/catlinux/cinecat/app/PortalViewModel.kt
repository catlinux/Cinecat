package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.data.LibraryRepository
import io.github.catlinux.cinecat.core.data.PortalData
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

sealed interface PortalUiState {
    data object Loading : PortalUiState
    data class Error(val message: String? = null) : PortalUiState
    data object Empty : PortalUiState
    data class Ready(
        val data: PortalData,
        val refreshing: Boolean = false
    ) : PortalUiState
}

class PortalViewModel(
    private val loadPortal: suspend () -> ApiResult<PortalData>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(repository: LibraryRepository, scope: CoroutineScope) : this(
        loadPortal = repository::portal,
        scope = scope
    )

    private val _state = MutableStateFlow<PortalUiState>(PortalUiState.Loading)
    val state: StateFlow<PortalUiState> = _state.asStateFlow()

    private var job: Job? = null

    init {
        refresh()
    }

    fun load() {
        retry()
    }

    fun retry() {
        job?.cancel()
        job = null
        _state.value = PortalUiState.Loading
        refresh()
    }

    fun refresh() {
        if (job?.isActive == true) {
            return
        }
        val current = _state.value
        if (current is PortalUiState.Ready) {
            _state.value = current.copy(refreshing = true)
        } else {
            _state.value = PortalUiState.Loading
        }
        job = scope.launch {
            val result = try {
                loadPortal()
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                ApiResult.NetworkError(IOException(error.message, error))
            }
            _state.value = when (result) {
                is ApiResult.Success -> {
                    if (result.value.rows.isEmpty()) {
                        PortalUiState.Empty
                    } else {
                        PortalUiState.Ready(result.value)
                    }
                }
                is ApiResult.HttpError -> PortalUiState.Error(result.message)
                is ApiResult.NetworkError -> PortalUiState.Error(result.cause.message)
            }
        }
    }

    override fun onCleared() {
        job?.cancel()
    }
}
