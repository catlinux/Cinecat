package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.MediaItem
import java.util.Locale

enum class DownloadLabel {
    Download,
    Preparing,
    Waiting,
    Downloading,
    Downloaded,
    Failed
}

fun downloadLabel(
    recordExists: Boolean,
    preparing: Boolean,
    phase: DownloadPhase?,
    waitingForServer: Boolean = false
): DownloadLabel = when {
    preparing -> DownloadLabel.Preparing
    !recordExists -> DownloadLabel.Download
    phase == DownloadPhase.Failed -> DownloadLabel.Failed
    phase == DownloadPhase.Completed -> DownloadLabel.Downloaded
    waitingForServer -> DownloadLabel.Waiting
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
