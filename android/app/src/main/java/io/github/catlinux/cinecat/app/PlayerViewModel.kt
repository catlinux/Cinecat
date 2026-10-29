package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.PlayResponse
import io.github.catlinux.cinecat.core.data.PlaybackRepository
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import okhttp3.HttpUrl

data class PlayerSubtitle(
    val url: String,
    val language: String,
    val label: String,
    val selected: Boolean
)

sealed interface PlayerUiState {
    data object Loading : PlayerUiState
    data class Error(val message: String? = null) : PlayerUiState
    data class Ready(
        val streamUrl: String,
        val isHls: Boolean,
        val subtitles: List<PlayerSubtitle>,
        val initialPositionSeconds: Double
    ) : PlayerUiState
}

class PlayerViewModel(
    val itemId: Long,
    val startSeconds: Double,
    private val caps: String,
    private val loadPlayback: suspend (caps: String?) -> ApiResult<PlayResponse>,
    private val persistProgress: suspend (position: Double, duration: Double) -> ApiResult<Unit>,
    private val serverUrl: () -> HttpUrl?,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(
        itemId: Long,
        startSeconds: Double,
        repository: PlaybackRepository,
        caps: String,
        scope: CoroutineScope,
        serverUrl: () -> HttpUrl? = { null }
    ) : this(
        itemId = itemId,
        startSeconds = startSeconds,
        caps = caps,
        loadPlayback = { requestedCaps -> repository.play(itemId, caps = requestedCaps) },
        persistProgress = { position, duration ->
            repository.saveProgress(itemId, position, duration)
        },
        serverUrl = serverUrl,
        scope = scope
    )

    private val _state = MutableStateFlow<PlayerUiState>(PlayerUiState.Loading)
    val state: StateFlow<PlayerUiState> = _state.asStateFlow()

    private var loadJob: Job? = null
    private var fallbackAttempted = false

    init {
        load()
    }

    fun load() {
        loadJob?.cancel()
        fallbackAttempted = false
        _state.value = PlayerUiState.Loading
        loadJob = scope.launch {
            fetch(useCaps = true, position = startSeconds.coerceAtLeast(0.0))
        }
    }

    fun retry() {
        load()
    }

    fun onPlaybackError(positionSeconds: Double = 0.0) {
        val ready = _state.value as? PlayerUiState.Ready ?: return
        if (ready.isHls || fallbackAttempted) {
            _state.value = PlayerUiState.Error()
            return
        }
        fallbackAttempted = true
        _state.value = PlayerUiState.Loading
        loadJob?.cancel()
        loadJob = scope.launch {
            fetch(
                useCaps = false,
                position = positionSeconds.coerceAtLeast(0.0)
            )
        }
    }

    fun saveProgress(position: Double, duration: Double) {
        if (!position.isFinite() || !duration.isFinite() || duration <= 0.0) {
            return
        }
        val safePosition = position.coerceAtLeast(0.0)
        scope.launch {
            safe { persistProgress(safePosition, duration) }
        }
    }

    private suspend fun fetch(useCaps: Boolean, position: Double) {
        when (val result = safe { loadPlayback(if (useCaps) caps else null) }) {
            is ApiResult.Success -> {
                val response = result.value
                _state.value = resolve(response, position)
            }
            is ApiResult.HttpError, is ApiResult.NetworkError -> {
                _state.value = PlayerUiState.Error()
            }
        }
    }

    private fun resolve(response: PlayResponse, position: Double): PlayerUiState {
        val origin = runCatching { serverUrl() }.getOrNull()
        val streamUrl = absoluteUrl(response.url, origin)
            ?: return PlayerUiState.Error()
        val subtitles = response.subtitles.mapNotNull { subtitle ->
            val url = absoluteUrl(subtitle.url, origin) ?: return@mapNotNull null
            PlayerSubtitle(
                url = url,
                language = subtitle.language,
                label = subtitle.title.ifBlank {
                    subtitle.languageName.ifBlank { subtitle.language }
                },
                selected = subtitle.id == response.selectedSubtitle
            )
        }
        return PlayerUiState.Ready(
            streamUrl = streamUrl,
            isHls = response.mode == "remux",
            subtitles = subtitles,
            initialPositionSeconds = position.coerceAtLeast(0.0)
        )
    }

    private fun absoluteUrl(value: String, origin: HttpUrl?): String? {
        if (value.isBlank()) {
            return null
        }
        if (value.startsWith("http://") || value.startsWith("https://")) {
            return value
        }
        return origin?.resolve(value)?.toString()
    }

    private suspend fun <T> safe(block: suspend () -> ApiResult<T>): ApiResult<T> {
        return try {
            block()
        } catch (error: CancellationException) {
            throw error
        } catch (error: Exception) {
            ApiResult.NetworkError(IOException(error.message, error))
        }
    }

    override fun onCleared() {
        loadJob?.cancel()
    }
}
