package io.github.catlinux.cinecat.app

import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.common.BitMatrix
import com.google.zxing.qrcode.QRCodeWriter
import com.google.zxing.qrcode.decoder.ErrorCorrectionLevel

object QrCode {
    private const val MARGIN_MODULES = 4

    fun matrix(content: String, sizePx: Int = 720): BitMatrix {
        require(content.isNotEmpty()) { "El contenido del QR no puede estar vacío" }
        require(sizePx > 0) { "El tamaño del QR debe ser positivo" }
        val hints = mapOf(
            EncodeHintType.CHARACTER_SET to "UTF-8",
            EncodeHintType.ERROR_CORRECTION to ErrorCorrectionLevel.M,
            EncodeHintType.MARGIN to MARGIN_MODULES
        )
        return QRCodeWriter().encode(content, BarcodeFormat.QR_CODE, sizePx, sizePx, hints)
    }
}
