package io.github.catlinux.cinecat.app

import com.google.zxing.BinaryBitmap
import com.google.zxing.LuminanceSource
import com.google.zxing.common.BitMatrix
import com.google.zxing.common.HybridBinarizer
import com.google.zxing.qrcode.QRCodeReader
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class QrCodeTest {
    private val url = "https://cinecat.example.com/vincular?codigo=123456"

    @Test
    fun qrMatrixHasQuietZoneAndEncodesTheUrl() {
        val matrix = QrCode.matrix(url, 320)

        assertEquals(320, matrix.width)
        assertEquals(320, matrix.height)
        assertTrue(hasDarkModules(matrix))
        assertFalse(matrix.get(0, 0))
        assertFalse(matrix.get(matrix.width - 1, matrix.height - 1))

        val decoded = QRCodeReader().decode(
            BinaryBitmap(HybridBinarizer(BitMatrixLuminanceSource(matrix)))
        )
        assertEquals(url, decoded.text)
    }

    private fun hasDarkModules(matrix: BitMatrix): Boolean {
        for (y in 0 until matrix.height) {
            for (x in 0 until matrix.width) {
                if (matrix.get(x, y)) {
                    return true
                }
            }
        }
        return false
    }

    private class BitMatrixLuminanceSource(
        private val matrix: BitMatrix
    ) : LuminanceSource(matrix.width, matrix.height) {
        override fun getRow(y: Int, row: ByteArray?): ByteArray {
            val output = if (row != null && row.size >= width) row else ByteArray(width)
            for (x in 0 until width) {
                output[x] = if (matrix.get(x, y)) 0 else -1
            }
            return output
        }

        override fun getMatrix(): ByteArray {
            val output = ByteArray(width * height)
            for (y in 0 until height) {
                for (x in 0 until width) {
                    output[y * width + x] = if (matrix.get(x, y)) 0 else -1
                }
            }
            return output
        }
    }
}
