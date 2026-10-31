package io.github.catlinux.cinecat.app

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/** Relación de aspecto que se le pasa a la ventana de imagen en imagen. */
data class PipAspectRatio(val width: Int, val height: Int) {
    companion object {
        val Default = PipAspectRatio(16, 9)
    }
}

/**
 * La imagen en imagen solo tiene sentido con un vídeo cargado, reproduciéndose,
 * fuera de la TV y en un dispositivo que la soporte.
 */
fun isPipEligible(
    hasVideo: Boolean,
    isPlaying: Boolean,
    isTelevision: Boolean,
    supportsPip: Boolean
): Boolean = hasVideo && isPlaying && !isTelevision && supportsPip

/**
 * Calcula la relación de aspecto del vídeo y la reduce. Si el tamaño todavía no
 * se conoce o cae fuera de los límites que admite Android, usa 16:9.
 */
fun pipAspectRatio(videoWidth: Int, videoHeight: Int): PipAspectRatio {
    if (videoWidth <= 0 || videoHeight <= 0) {
        return PipAspectRatio.Default
    }
    val ratio = videoWidth.toDouble() / videoHeight.toDouble()
    if (ratio < 1.0 / 2.39 || ratio > 2.39) {
        return PipAspectRatio.Default
    }
    val divisor = greatestCommonDivisor(videoWidth, videoHeight)
    return PipAspectRatio(videoWidth / divisor, videoHeight / divisor)
}

private tailrec fun greatestCommonDivisor(left: Int, right: Int): Int =
    if (right == 0) left else greatestCommonDivisor(right, left % right)

/**
 * Estado compartido entre el reproductor y la actividad. PlayerScreen lo
 * actualiza; MainActivity lo usa para entrar en PiP y PlayerScreen para
 * esconder los controles.
 */
class PictureInPictureController {
    var eligible by mutableStateOf(false)
    var inPip by mutableStateOf(false)
    var aspectRatio by mutableStateOf(PipAspectRatio.Default)

    internal var wasInPip = false
    internal var onStopRequested: (() -> Unit)? = null

    fun requestStop() {
        onStopRequested?.invoke()
    }
}
