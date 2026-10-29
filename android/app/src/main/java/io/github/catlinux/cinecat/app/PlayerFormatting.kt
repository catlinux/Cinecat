package io.github.catlinux.cinecat.app

import java.util.Locale

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
