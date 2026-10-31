package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ItemPreferences
import io.github.catlinux.cinecat.core.api.PlayResponse

data class ResolvedPlayerPreferences(
    val audioIndex: Int,
    val subtitleId: String?,
    val quality: String
)

/**
 * Aplica las preferencias guardadas solo si siguen existiendo en la respuesta
 * actual. Si no existen, conserva la selección que ya trae el servidor.
 */
fun resolvePlayerPreferences(
    play: PlayResponse,
    preferences: ItemPreferences?
): ResolvedPlayerPreferences {
    val audioIndex = preferences?.audioIndex
        ?.takeIf { index -> play.audioTracks.any { it.index == index } }
        ?: play.selectedAudio

    val subtitleId = when (val preferred = preferences?.subtitleId) {
        null -> play.selectedSubtitle
        "" -> null
        else -> if (play.subtitles.any { it.id == preferred }) preferred else play.selectedSubtitle
    }

    val quality = preferences?.quality
        ?.takeIf { value -> play.qualities.any { it.value == value } }
        ?: play.selectedQuality

    return ResolvedPlayerPreferences(
        audioIndex = audioIndex,
        subtitleId = subtitleId,
        quality = quality
    )
}
