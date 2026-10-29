package io.github.catlinux.cinecat.app

fun formatRuntime(minutes: Int?): String {
    if (minutes == null || minutes <= 0) {
        return ""
    }
    val hours = minutes / 60
    val remaining = minutes % 60
    return when {
        hours == 0 -> "$remaining min"
        remaining == 0 -> "$hours h"
        else -> "$hours h $remaining min"
    }
}
