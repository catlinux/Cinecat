package io.github.catlinux.cinecat.app

import org.junit.Assert.assertEquals
import org.junit.Test

class PlayerFormattingTest {
    @Test
    fun formatsHoursAndMinutes() {
        assertEquals("0:00", formatPlaybackPosition(0.0))
        assertEquals("0:42", formatPlaybackPosition(42.9))
        assertEquals("2:05", formatPlaybackPosition(125.0))
        assertEquals("1:02:15", formatPlaybackPosition(3735.0))
    }
}
