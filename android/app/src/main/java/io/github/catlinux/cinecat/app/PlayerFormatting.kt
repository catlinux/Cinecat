package io.github.catlinux.cinecat.app

import java.util.Locale

enum class PlayerTrackKind {
    Audio,
    Subtitle
}

fun formatPlayerTrackLabel(
    label: String?,
    language: String?,
    kind: PlayerTrackKind,
    position: Int,
    audioFormat: String,
    subtitleFormat: String,
    // La app solo tiene recursos en castellano.
    locale: Locale = Locale.forLanguageTag("es")
): String {
    label?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }

    val tag = language?.trim().orEmpty()
    if (tag.isNotEmpty() && !tag.equals("und", ignoreCase = true)) {
        val parsed = Locale.forLanguageTag(tag)
        val name = parsed.getDisplayLanguage(locale).trim()
        if (name.isNotEmpty() &&
            !name.equals(tag, ignoreCase = true) &&
            !name.equals("und", ignoreCase = true)
        ) {
            return name.replaceFirstChar { char ->
                if (char.isLowerCase()) char.titlecase(locale) else char.toString()
            }
        }
    }

    val template = when (kind) {
        PlayerTrackKind.Audio -> audioFormat
        PlayerTrackKind.Subtitle -> subtitleFormat
    }
    return String.format(locale, template, position.coerceAtLeast(1))
}

fun formatPlaybackPosition(seconds: Double): String {
    val total = if (seconds.isFinite()) seconds.toLong().coerceAtLeast(0) else 0
    val hours = total / 3600
    val minutes = (total % 3600) / 60
    val remaining = total % 60
    return if (hours > 0) {
        String.format(Locale.US, "%d:%02d:%02d", hours, minutes, remaining)
    } else {
        String.format(Locale.US, "%d:%02d", minutes, remaining)
    }
}
