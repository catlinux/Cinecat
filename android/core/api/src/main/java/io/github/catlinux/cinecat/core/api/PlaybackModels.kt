package io.github.catlinux.cinecat.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class PlayResponse(
    val url: String,
    val mode: String,
    val reason: String? = null,
    @SerialName("expires_at") val expiresAt: String = "",
    @SerialName("audio_tracks") val audioTracks: List<AudioTrack> = emptyList(),
    @SerialName("selected_audio") val selectedAudio: Int = 0,
    val qualities: List<QualityOption> = emptyList(),
    @SerialName("selected_quality") val selectedQuality: String = "",
    val subtitles: List<SubtitleOption> = emptyList(),
    @SerialName("selected_subtitle") val selectedSubtitle: String? = null
)

@Serializable
data class AudioTrack(
    val index: Int = 0,
    val language: String = "",
    @SerialName("language_name") val languageName: String = "",
    val codec: String = "",
    val channels: Int = 0,
    @SerialName("channel_layout") val channelLayout: String = "",
    val title: String = "",
    val default: Boolean = false
) {
    val isDefault: Boolean get() = default
}

@Serializable
data class QualityOption(
    val value: String = "",
    val label: String = ""
)

@Serializable
data class SubtitleOption(
    val id: String = "",
    val source: String = "",
    val language: String = "",
    @SerialName("language_name") val languageName: String = "",
    val title: String = "",
    val codec: String = "",
    val forced: Boolean = false,
    val default: Boolean = false,
    val url: String = ""
) {
    val isDefault: Boolean get() = default
}

@Serializable
data class ProgressRequest(
    // Sin valores por defecto: kotlinx omitiría una posición 0 y el servidor exige ambos campos.
    @SerialName("position_seconds") val positionSeconds: Double,
    @SerialName("duration_seconds") val durationSeconds: Double
)

@Serializable
data class ItemPreferences(
    @SerialName("audio_index") val audioIndex: Int? = null,
    @SerialName("subtitle_id") val subtitleId: String? = null,
    val quality: String? = null
)

@Serializable
data class PreferencesPatch(
    @SerialName("audio_index") val audioIndex: Int? = null,
    @SerialName("subtitle_id") val subtitleId: String? = null,
    val quality: String? = null
)
