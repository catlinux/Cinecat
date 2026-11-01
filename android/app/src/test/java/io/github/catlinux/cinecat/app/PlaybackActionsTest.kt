package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.PlaybackProgress
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PlaybackActionsTest {
    @Test
    fun offersStartFromBeginningOnlyWithRealProgress() {
        assertFalse(shouldOfferPlayFromStart(0.0, 1000.0))
        assertFalse(shouldOfferPlayFromStart(-10.0, 1000.0))
        assertTrue(shouldOfferPlayFromStart(10.0, 1000.0))
    }

    @Test
    fun hidesOptionNearTheEnd() {
        assertTrue(shouldOfferPlayFromStart(949.0, 1000.0))
        assertFalse(shouldOfferPlayFromStart(950.0, 1000.0))
        assertFalse(shouldOfferPlayFromStart(999.0, 1000.0))
    }

    @Test
    fun keepsOptionWhenDurationIsUnknown() {
        assertTrue(shouldOfferPlayFromStart(120.0, 0.0))
        assertTrue(shouldOfferPlayFromStart(120.0, Double.NaN))
        assertFalse(shouldOfferPlayFromStart(Double.NaN, 1000.0))
    }

    @Test
    fun readsPlaybackProgressSafely() {
        assertFalse(shouldOfferPlayFromStart(null as PlaybackProgress?))
        assertFalse(
            shouldOfferPlayFromStart(
                PlaybackProgress(positionSeconds = 0.0, durationSeconds = 1000.0)
            )
        )
        assertTrue(
            shouldOfferPlayFromStart(
                PlaybackProgress(positionSeconds = 100.0, durationSeconds = 1000.0)
            )
        )
    }
}
