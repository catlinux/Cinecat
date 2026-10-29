package io.github.catlinux.cinecat.app

import androidx.core.graphics.createBitmap
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap

private const val QR_DARK = 0xFF3A1013.toInt()
private const val QR_LIGHT = 0xFFFBF7EE.toInt()

fun qrCodeImage(content: String, sizePx: Int = 720): ImageBitmap {
    val matrix = QrCode.matrix(content, sizePx)
    val pixels = IntArray(matrix.width * matrix.height)
    for (y in 0 until matrix.height) {
        for (x in 0 until matrix.width) {
            pixels[y * matrix.width + x] = if (matrix.get(x, y)) QR_DARK else QR_LIGHT
        }
    }
    val bitmap = createBitmap(matrix.width, matrix.height)
    bitmap.setPixels(pixels, 0, matrix.width, 0, 0, matrix.width, matrix.height)
    return bitmap.asImageBitmap()
}
