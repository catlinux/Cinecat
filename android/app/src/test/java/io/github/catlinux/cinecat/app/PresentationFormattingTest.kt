package io.github.catlinux.cinecat.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PresentationFormattingTest {
    @Test
    fun episodeCodeUsesTwoDigits() {
        assertEquals("S01E02", formatEpisodeCode(1, 2))
        assertEquals("S10E24", formatEpisodeCode(10, 24))
        assertEquals("", formatEpisodeCode(null, null))
    }

    @Test
    fun navigationUsesRailFromSixHundredDp() {
        assertFalse(useNavigationRail(599))
        assertTrue(useNavigationRail(600))
        assertTrue(useNavigationRail(840))
    }
}
