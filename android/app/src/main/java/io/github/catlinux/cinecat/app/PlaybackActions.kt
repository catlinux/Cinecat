package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.PlaybackProgress

/**
 * Ofrece empezar de cero cuando hay progreso real pero no se está casi al
 * final; en ese último tramo el botón de continuar es suficiente.
 */
fun shouldOfferPlayFromStart(
    positionSeconds: Double,
    durationSeconds: Double
): Boolean {
    if (!positionSeconds.isFinite() || positionSeconds <= 0.0) {
        return false
    }
    if (durationSeconds.isFinite() && durationSeconds > 0.0 &&
        positionSeconds >= durationSeconds * 0.95
    ) {
        return false
    }
    return true
}

fun shouldOfferPlayFromStart(progress: PlaybackProgress?): Boolean {
    return progress != null &&
        shouldOfferPlayFromStart(progress.positionSeconds, progress.durationSeconds)
}
