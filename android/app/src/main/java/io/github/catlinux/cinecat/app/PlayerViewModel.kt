package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.ItemPreferences
import io.github.catlinux.cinecat.core.api.PlayResponse
import io.github.catlinux.cinecat.core.api.PreferencesPatch
import io.github.catlinux.cinecat.core.api.SubtitleOption
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
    val selected: Boolean,
    val id: String = ""
)

data class PlayerBurnSubtitle(
    val id: String,
    val label: String,
    val selected: Boolean
)

data class PlaybackRequest(
    val caps: String? = null,
    val audio: Int? = null,
    val quality: String? = null,
    val subtitle: String? = null
)

sealed interface PlayerUiState {
    data object Loading : PlayerUiState
    data class Error(val message: String? = null) : PlayerUiState
    data class Ready(
        val streamUrl: String,
        val isHls: Boolean,
        val subtitles: List<PlayerSubtitle>,
        val initialPositionSeconds: Double,
        val selectedAudio: Int = 0,
        val selectedSubtitle: String? = null,
        val selectedQuality: String = "",
        val burnSubtitles: List<PlayerBurnSubtitle> = emptyList()
    ) : PlayerUiState
}

fun shouldReloadSubtitle(
    selectedSubtitle: String?,
    targetSubtitle: String?,
    burnSubtitles: List<PlayerBurnSubtitle>
): Boolean {
    if (selectedSubtitle == targetSubtitle) {
        return false
    }
    val currentIsImage = selectedSubtitle != null &&
        burnSubtitles.any { it.id == selectedSubtitle }
    val targetIsImage = targetSubtitle != null &&
        burnSubtitles.any { it.id == targetSubtitle }
    return currentIsImage || targetIsImage
}

class PlayerViewModel(
    val itemId: Long,
    val startSeconds: Double,
    private val caps: String,
    private val loadPlayback: suspend (request: PlaybackRequest) -> ApiResult<PlayResponse>,
    private val persistProgress: suspend (position: Double, duration: Double) -> ApiResult<Unit>,
    private val loadPreferences: suspend () -> ApiResult<ItemPreferences> = {
        ApiResult.Success(ItemPreferences())
    },
    private val persistPreferences: suspend (patch: PreferencesPatch) -> ApiResult<Unit> = {
        ApiResult.Success(Unit)
    },
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
        loadPlayback = { request ->
            repository.play(
                itemId,
                caps = request.caps,
                audio = request.audio,
                quality = request.quality,
                subtitle = request.subtitle
            )
        },
        persistProgress = { position, duration ->
            repository.saveProgress(itemId, position, duration)
        },
        loadPreferences = { repository.itemPreferences(itemId) },
        persistPreferences = { patch -> repository.saveItemPreferences(itemId, patch) },
        serverUrl = serverUrl,
        scope = scope
    )

    private val _state = MutableStateFlow<PlayerUiState>(PlayerUiState.Loading)
    val state: StateFlow<PlayerUiState> = _state.asStateFlow()

    private var loadJob: Job? = null
    private var fallbackAttempted = false
    private var initialPreferences: ItemPreferences? = null

    init {
        load()
    }

    fun load() {
        loadJob?.cancel()
        fallbackAttempted = false
        _state.value = PlayerUiState.Loading
        loadJob = scope.launch {
            initialPreferences = preferencesOrNull()
            fetch(
                useCaps = true,
                position = startSeconds.coerceAtLeast(0.0),
                preferences = initialPreferences
            )
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
                position = positionSeconds.coerceAtLeast(0.0),
                preferences = initialPreferences
            )
        }
    }

    fun savePlaybackPreferences(
        audioIndex: Int? = null,
        subtitleId: String? = null,
        quality: String? = null
    ) {
        if (audioIndex == null && subtitleId == null && quality == null) {
            return
        }
        scope.launch {
            safe {
                persistPreferences(
                    PreferencesPatch(
                        audioIndex = audioIndex,
                        subtitleId = subtitleId,
                        quality = quality
                    )
                )
            }
        }
    }

    fun selectSubtitle(subtitleId: String?, positionSeconds: Double) {
        val ready = _state.value as? PlayerUiState.Ready ?: return
        val target = subtitleId?.takeIf { it.isNotBlank() }
        val currentIsImage = ready.burnSubtitles.any { it.selected }
        val targetIsImage = target != null && ready.burnSubtitles.any { it.id == target }
        val savedId = subtitleId ?: ""
        val quality = ready.selectedQuality.takeIf { it.isNotBlank() }

        if (!shouldReloadSubtitle(ready.selectedSubtitle, target, ready.burnSubtitles)) {
            savePlaybackPreferences(subtitleId = savedId, quality = quality)
            return
        }

        // Al salir de una pista quemada, el texto elegido se aplica en el nuevo
        // medio para que la petición pueda pedir subtitle vacío y quitar el quemado.
        val pendingTextSubtitle = target?.takeIf { currentIsImage && !targetIsImage }
        val requestSubtitle = if (pendingTextSubtitle != null) "" else savedId

        loadJob?.cancel()
        _state.value = PlayerUiState.Loading
        loadJob = scope.launch {
            safe {
                persistPreferences(PreferencesPatch(subtitleId = savedId, quality = quality))
            }
            fetch(
                useCaps = !fallbackAttempted,
                position = positionSeconds.coerceAtLeast(0.0),
                preferences = null,
                audio = ready.selectedAudio,
                quality = quality,
                subtitle = requestSubtitle,
                pendingTextSubtitle = pendingTextSubtitle
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

    private suspend fun fetch(
        useCaps: Boolean,
        position: Double,
        preferences: ItemPreferences?,
        audio: Int? = null,
        quality: String? = null,
        subtitle: String? = null,
        pendingTextSubtitle: String? = null
    ) {
        val request = PlaybackRequest(
            caps = if (useCaps) caps else null,
            audio = audio,
            quality = quality,
            subtitle = subtitle
        )
        when (val result = safe { loadPlayback(request) }) {
            is ApiResult.Success -> {
                val resolved = resolve(result.value, position, preferences)
                _state.value = pendingTextSubtitle
                    ?.let { resolved.withSelectedTextSubtitle(it) }
                    ?: resolved
            }
            is ApiResult.HttpError, is ApiResult.NetworkError -> {
                _state.value = PlayerUiState.Error()
            }
        }
    }

    private suspend fun preferencesOrNull(): ItemPreferences? {
        return when (val result = safe { loadPreferences() }) {
            is ApiResult.Success -> result.value
            is ApiResult.HttpError, is ApiResult.NetworkError -> null
        }
    }

    private fun resolve(
        response: PlayResponse,
        position: Double,
        preferences: ItemPreferences?
    ): PlayerUiState {
        val origin = runCatching { serverUrl() }.getOrNull()
        val streamUrl = absoluteUrl(response.url, origin)
            ?: return PlayerUiState.Error()
        val resolved = resolvePlayerPreferences(response, preferences)
        val subtitles = mutableListOf<PlayerSubtitle>()
        val burnSubtitles = mutableListOf<PlayerBurnSubtitle>()
        response.subtitles.forEach { subtitle ->
            val label = subtitleLabel(subtitle)
            if (subtitle.burnIn) {
                burnSubtitles += PlayerBurnSubtitle(
                    id = subtitle.id,
                    label = "$label · imagen",
                    selected = subtitle.id == resolved.subtitleId
                )
                return@forEach
            }
            val url = absoluteUrl(subtitle.url, origin) ?: return@forEach
            subtitles += PlayerSubtitle(
                id = subtitle.id,
                url = url,
                language = subtitle.language,
                label = label,
                selected = subtitle.id == resolved.subtitleId
            )
        }
        return PlayerUiState.Ready(
            streamUrl = streamUrl,
            isHls = response.mode == "remux",
            subtitles = subtitles,
            initialPositionSeconds = position.coerceAtLeast(0.0),
            selectedAudio = resolved.audioIndex,
            selectedSubtitle = resolved.subtitleId,
            selectedQuality = resolved.quality,
            burnSubtitles = burnSubtitles
        )
    }

    private fun subtitleLabel(subtitle: SubtitleOption): String {
        return subtitle.title.ifBlank {
            subtitle.languageName.ifBlank { subtitle.language }
        }
    }

    private fun PlayerUiState.withSelectedTextSubtitle(id: String): PlayerUiState {
        val ready = this as? PlayerUiState.Ready ?: return this
        if (ready.subtitles.none { it.id == id }) {
            return this
        }
        return ready.copy(
            selectedSubtitle = id,
            subtitles = ready.subtitles.map { it.copy(selected = it.id == id) },
            burnSubtitles = ready.burnSubtitles.map { it.copy(selected = false) }
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
