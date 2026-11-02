package io.github.catlinux.cinecat.app

val PROFILE_MAX_QUALITIES = listOf("", "480", "720", "1080")

val PROFILE_AUDIO_LANGUAGES = listOf(
    "",
    "es",
    "ca",
    "en",
    "fr",
    "de",
    "it",
    "pt",
    "ja"
)

val PROFILE_SUBTITLE_LANGUAGES = listOf(
    "",
    "off",
    "es",
    "ca",
    "en",
    "fr",
    "de",
    "it",
    "pt",
    "ja"
)

// Un valor guardado fuera de la lista se añade al final para que el ciclo lo
// conserve en lugar de perderlo al primer toque.
fun nextProfileValue(current: String, options: List<String>): String {
    val values = if (current.isNotEmpty() && current !in options) {
        options + current
    } else {
        options
    }
    if (values.isEmpty()) {
        return ""
    }
    val index = values.indexOf(current)
    return values[(index + 1) % values.size]
}
