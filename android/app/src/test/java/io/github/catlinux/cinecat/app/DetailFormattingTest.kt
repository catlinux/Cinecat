package io.github.catlinux.cinecat.app

import org.junit.Assert.assertEquals
import org.junit.Test

class DetailFormattingTest {
    @Test
    fun formatsRuntime() {
        assertEquals("45 min", formatRuntime(45))
        assertEquals("1 h", formatRuntime(60))
        assertEquals("1 h 52 min", formatRuntime(112))
        assertEquals("", formatRuntime(null))
        assertEquals("", formatRuntime(0))
    }
}
