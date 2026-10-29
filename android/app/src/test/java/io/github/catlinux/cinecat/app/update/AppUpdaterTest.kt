package io.github.catlinux.cinecat.app.update

import java.io.ByteArrayInputStream
import java.io.File
import java.security.MessageDigest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AppUpdaterTest {
    @Test
    fun allowsOnlyTheReleaseDownloadPrefix() {
        assertTrue(
            isAllowedDownloadUrl(
                "https://github.com/catlinux/Cinecat/releases/download/v0.34.1/cinecat-0.34.1.apk"
            )
        )
        assertFalse(isAllowedDownloadUrl("http://github.com/catlinux/Cinecat/releases/download/v1/a.apk"))
        assertFalse(isAllowedDownloadUrl("https://github.com/catlinux/Cinecat/releases/download/v1/../a.apk"))
        assertFalse(isAllowedDownloadUrl("https://github.com/catlinux/Cinecat/releases/download/v1/a@b.apk"))
        assertFalse(isAllowedDownloadUrl("https://github.com/catlinux/Cinecat/releases/download/v1/a.apk?x=1"))
        assertFalse(isAllowedDownloadUrl("https://github.com/otro/Cinecat/releases/download/v1/a.apk"))
        assertFalse(isAllowedDownloadUrl("https://github.com/catlinux/Cinecat/releases/download"))
    }

    @Test
    fun checksEveryTwelveHours() {
        val now = 1_000_000_000_000L
        assertTrue(shouldCheck(0L, now))
        assertFalse(shouldCheck(now, now))
        assertFalse(shouldCheck(now - UPDATE_CHECK_INTERVAL_MILLIS + 1, now))
        assertTrue(shouldCheck(now - UPDATE_CHECK_INTERVAL_MILLIS, now))
    }

    @Test
    fun copiesAndVerifiesTheChecksum() {
        val content = "contenido del apk".toByteArray()
        val target = File.createTempFile("cinecat-update", ".apk").apply { delete() }

        val failure = copyWithSha256(
            input = ByteArrayInputStream(content),
            target = target,
            expectedSha256 = sha256(content)
        )

        assertNull(failure)
        assertEquals(content.toList(), target.readBytes().toList())
        target.delete()
    }

    @Test
    fun removesTheFileWhenTheChecksumDoesNotMatch() {
        val target = File.createTempFile("cinecat-update", ".apk").apply { delete() }
        val failure = copyWithSha256(
            input = ByteArrayInputStream("contenido".toByteArray()),
            target = target,
            expectedSha256 = sha256("otro".toByteArray())
        )

        assertEquals(UpdateFailure.HashMismatch, failure)
        assertFalse(target.exists())
    }

    @Test
    fun removesTheFileWhenItExceedsTheLimit() {
        val target = File.createTempFile("cinecat-update", ".apk").apply { delete() }
        val content = ByteArray(11) { it.toByte() }
        val failure = copyWithSha256(
            input = ByteArrayInputStream(content),
            target = target,
            expectedSha256 = sha256(content),
            maxBytes = 10
        )

        assertEquals(UpdateFailure.TooLarge, failure)
        assertFalse(target.exists())
    }

    private fun sha256(bytes: ByteArray): String {
        return MessageDigest.getInstance("SHA-256").digest(bytes).joinToString("") {
            "%02x".format(it)
        }
    }
}
