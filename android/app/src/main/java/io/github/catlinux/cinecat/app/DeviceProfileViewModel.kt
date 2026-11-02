package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.SessionProfile
import io.github.catlinux.cinecat.core.data.DeviceProfileRepository
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

sealed interface DeviceProfileUiState {
    data object Loading : DeviceProfileUiState
    data object Unsupported : DeviceProfileUiState
    data class Error(val message: String? = null) : DeviceProfileUiState
    data class Ready(
        val profile: SessionProfile,
        val saving: Boolean = false,
        val saveResult: DeviceProfileSaveResult? = null
    ) : DeviceProfileUiState
}

enum class DeviceProfileSaveResult {
    Saved,
    Failed
}

class DeviceProfileViewModel(
    private val loadProfile: suspend () -> ApiResult<SessionProfile>,
    private val saveProfile: suspend (SessionProfile) -> ApiResult<SessionProfile>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(repository: DeviceProfileRepository, scope: CoroutineScope) : this(
        loadProfile = repository::currentProfile,
        saveProfile = repository::saveCurrentProfile,
        scope = scope
    )

    private val _state = MutableStateFlow<DeviceProfileUiState>(DeviceProfileUiState.Loading)
    val state: StateFlow<DeviceProfileUiState> = _state.asStateFlow()

    private var loadJob: Job? = null
    private var saveJob: Job? = null
    private var extraAudioLanguages = emptyList<String>()
    private var extraSubtitleLanguages = emptyList<String>()

    init {
        load()
    }

    fun load() {
        loadJob?.cancel()
        saveJob?.cancel()
        _state.value = DeviceProfileUiState.Loading
        loadJob = scope.launch {
            _state.value = when (val result = safe { loadProfile() }) {
                is ApiResult.Success -> {
                    setExtrasFrom(result.value)
                    DeviceProfileUiState.Ready(result.value)
                }
                is ApiResult.HttpError -> {
                    if (result.code == 404) {
                        DeviceProfileUiState.Unsupported
                    } else {
                        DeviceProfileUiState.Error(result.message)
                    }
                }
                is ApiResult.NetworkError -> DeviceProfileUiState.Error(result.cause.message)
            }
        }
    }

    fun retry() {
        load()
    }

    fun cycleMaxQuality() {
        updateProfile { it.copy(maxQuality = nextProfileValue(it.maxQuality, PROFILE_MAX_QUALITIES)) }
    }

    fun cycleAudioLanguage() {
        updateProfile {
            it.copy(
                audioLanguage = nextProfileValue(
                    it.audioLanguage,
                    PROFILE_AUDIO_LANGUAGES + extraAudioLanguages
                )
            )
        }
    }

    fun cycleSubtitleLanguage() {
        updateProfile {
            it.copy(
                subtitleLanguage = nextProfileValue(
                    it.subtitleLanguage,
                    PROFILE_SUBTITLE_LANGUAGES + extraSubtitleLanguages
                )
            )
        }
    }

    private fun updateProfile(transform: (SessionProfile) -> SessionProfile) {
        val ready = _state.value as? DeviceProfileUiState.Ready ?: return
        val next = transform(ready.profile)
        if (next == ready.profile) {
            return
        }
        saveJob?.cancel()
        _state.value = ready.copy(profile = next, saving = true, saveResult = null)
        saveJob = scope.launch {
            when (val result = safe { saveProfile(next) }) {
                is ApiResult.Success -> {
                    rememberExtras(result.value)
                    val current = _state.value as? DeviceProfileUiState.Ready ?: return@launch
                    _state.value = current.copy(
                        profile = result.value,
                        saving = false,
                        saveResult = DeviceProfileSaveResult.Saved
                    )
                }
                is ApiResult.HttpError -> {
                    if (result.code == 404) {
                        _state.value = DeviceProfileUiState.Unsupported
                    } else {
                        val current = _state.value as? DeviceProfileUiState.Ready ?: return@launch
                        _state.value = current.copy(
                            saving = false,
                            saveResult = DeviceProfileSaveResult.Failed
                        )
                    }
                }
                is ApiResult.NetworkError -> {
                    val current = _state.value as? DeviceProfileUiState.Ready ?: return@launch
                    _state.value = current.copy(
                        saving = false,
                        saveResult = DeviceProfileSaveResult.Failed
                    )
                }
            }
        }
    }

    private fun setExtrasFrom(profile: SessionProfile) {
        extraAudioLanguages = extraLanguage(profile.audioLanguage, PROFILE_AUDIO_LANGUAGES)
        extraSubtitleLanguages = extraLanguage(
            profile.subtitleLanguage,
            PROFILE_SUBTITLE_LANGUAGES
        )
    }

    private fun rememberExtras(profile: SessionProfile) {
        extraAudioLanguages = rememberExtraLanguage(
            extraAudioLanguages,
            profile.audioLanguage,
            PROFILE_AUDIO_LANGUAGES
        )
        extraSubtitleLanguages = rememberExtraLanguage(
            extraSubtitleLanguages,
            profile.subtitleLanguage,
            PROFILE_SUBTITLE_LANGUAGES
        )
    }

    private fun extraLanguage(value: String, options: List<String>): List<String> {
        return if (value.isNotEmpty() && value !in options) listOf(value) else emptyList()
    }

    private fun rememberExtraLanguage(
        current: List<String>,
        value: String,
        options: List<String>
    ): List<String> {
        return if (value.isNotEmpty() && value !in options && value !in current) {
            current + value
        } else {
            current
        }
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
        saveJob?.cancel()
    }
}
