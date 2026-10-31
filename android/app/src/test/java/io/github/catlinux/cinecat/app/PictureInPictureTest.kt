package io.github.catlinux.cinecat.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PictureInPictureTest {
    @Test
    fun onlyMobilesWithVideoPlayingAndSupportAreEligible() {
        assertTrue(isPipEligible(true, true, false, true))
        assertFalse(isPipEligible(false, true, false, true))
        assertFalse(isPipEligible(true, false, false, true))
        assertFalse(isPipEligible(true, true, true, true))
        assertFalse(isPipEligible(true, true, false, false))
    }

    @Test
    fun usesTheReducedVideoAspectRatio() {
        assertEquals(PipAspectRatio(16, 9), pipAspectRatio(1920, 1080))
        assertEquals(PipAspectRatio(16, 9), pipAspectRatio(3840, 2160))
        assertEquals(PipAspectRatio(4, 3), pipAspectRatio(1440, 1080))
        assertEquals(PipAspectRatio(9, 16), pipAspectRatio(1080, 1920))
    }

    @Test
    fun fallsBackToSixteenNineWhenTheSizeIsUnknownOrExtreme() {
        assertEquals(PipAspectRatio.Default, pipAspectRatio(0, 1080))
        assertEquals(PipAspectRatio.Default, pipAspectRatio(1920, 0))
        assertEquals(PipAspectRatio.Default, pipAspectRatio(-1920, 1080))
        assertEquals(PipAspectRatio.Default, pipAspectRatio(100, 1000))
        assertEquals(PipAspectRatio.Default, pipAspectRatio(3000, 1000))
    }
}
