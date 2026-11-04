package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.PlayResponse
import java.net.URLDecoder
import java.nio.charset.StandardCharsets
import java.util.Locale

enum class DownloadLabel {
    Download,
    Preparing,
    Downloading,
    Downloaded,
    Failed
}

// El servidor decide con la ficha del archivo; un servidor anterior no envía el
// campo y la descarga queda bloqueada, como antes.
fun canDownload(play: PlayResponse): Boolean = play.downloadDirect

enum class DownloadBlockedKind {
    Audio,
    Video,
    Container,
    Other
}

fun downloadBlockedKind(reason: String?): DownloadBlockedKind = when (reason) {
    "audio" -> DownloadBlockedKind.Audio
    "video" -> DownloadBlockedKind.Video
    "container" -> DownloadBlockedKind.Container
    else -> DownloadBlockedKind.Other
}

fun downloadLabel(
    recordExists: Boolean,
    preparing: Boolean,
    phase: DownloadPhase?
): DownloadLabel = when {
    preparing -> DownloadLabel.Preparing
    !recordExists -> DownloadLabel.Download
    phase == DownloadPhase.Failed -> DownloadLabel.Failed
    phase == DownloadPhase.Completed -> DownloadLabel.Downloaded
    else -> DownloadLabel.Downloading
}

fun downloadDisplayTitle(item: MediaItem): String {
    val fallback = item.title.ifBlank { item.fileTitle }.trim()
    if (item.kind != "episode") {
        return fallback
    }
    val series = item.seriesTitle?.takeIf { it.isNotBlank() } ?: fallback
    val code = downloadEpisodeCode(item.season, item.episode)
    val episode = item.episodeTitle?.takeIf { it.isNotBlank() } ?: fallback
    return listOf(series, code, episode)
        .filter { it.isNotBlank() }
        .distinct()
        .joinToString(" · ")
}

fun downloadEpisodeCode(season: Int?, episode: Int?): String {
    if (season == null && episode == null) {
        return ""
    }
    return "T${season?.toString() ?: "?"}E${episode?.toString() ?: "?"}"
}

fun downloadExtension(disposition: String?, contentType: String?): String {
    val name = filenameFromDisposition(disposition)
    val extension = name?.substringAfterLast('.', "")?.lowercase(Locale.ROOT).orEmpty()
    if (extension.matches(Regex("[a-z0-9]{1,8}"))) {
        return extension
    }
    return when (contentType?.substringBefore(';')?.trim()?.lowercase(Locale.ROOT)) {
        "video/x-matroska" -> "mkv"
        "video/mp4" -> "mp4"
        "video/quicktime" -> "mov"
        else -> "bin"
    }
}

fun downloadFileName(itemId: Long, disposition: String?, contentType: String?): String =
    "$itemId.${downloadExtension(disposition, contentType)}"

fun downloadPercent(downloaded: Long, total: Long): Int? {
    if (total <= 0L) {
        return null
    }
    return ((downloaded.toDouble() * 100.0) / total.toDouble())
        .coerceIn(0.0, 100.0)
        .toInt()
}

fun formatDownloadSize(bytes: Long): String {
    if (bytes <= 0L) {
        return ""
    }
    val units = listOf("B", "KB", "MB", "GB", "TB")
    var value = bytes.toDouble()
    var unit = 0
    while (value >= 1024.0 && unit < units.lastIndex) {
        value /= 1024.0
        unit++
    }
    return if (unit == 0) {
        "$bytes ${units[unit]}"
    } else {
        String.format(Locale.ROOT, "%.1f %s", value, units[unit])
    }
}

fun filenameFromDisposition(disposition: String?): String? {
    if (disposition.isNullOrBlank()) {
        return null
    }
    val extended = Regex("""filename\*\s*=\s*UTF-8''([^;]+)""", RegexOption.IGNORE_CASE)
        .find(disposition)
        ?.groupValues
        ?.getOrNull(1)
    val plain = Regex("""filename\s*=\s*"?([^";]+)"?""", RegexOption.IGNORE_CASE)
        .find(disposition)
        ?.groupValues
        ?.getOrNull(1)
    val raw = extended ?: plain ?: return null
    return runCatching {
        URLDecoder.decode(raw.replace("+", "%2B"), StandardCharsets.UTF_8.name())
    }.getOrDefault(raw)
}
