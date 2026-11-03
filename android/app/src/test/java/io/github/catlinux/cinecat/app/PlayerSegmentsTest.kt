package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.NextEpisode
import io.github.catlinux.cinecat.core.api.PlaySegment
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PlayerSegmentsTest {
    private val segments = listOf(
        PlaySegment(kind = "intro", start = 10.0, end = 20.0, source = "chapter"),
        PlaySegment(kind = "credits", start = 80.0, end = 90.0, source = "manual")
    )

    @Test
    fun introButtonCoversStartInclusivelyAndEndExclusively() {
        assertNull(activeSegmentButton(9.999, segments))
        val atStart = activeSegmentButton(10.0, segments)
        assertEquals(SegmentButtonKind.Intro, atStart?.kind)
        assertEquals(R.string.player_skip_intro, atStart?.labelRes)
        assertEquals(20.0, atStart?.endSeconds ?: 0.0, 0.0)

        assertEquals(SegmentButtonKind.Intro, activeSegmentButton(19.999, segments)?.kind)
        assertNull(activeSegmentButton(20.0, segments))
    }

    @Test
    fun creditsButtonUsesItsOwnTextAndEnd() {
        val button = activeSegmentButton(80.0, segments)
        assertEquals(SegmentButtonKind.Credits, button?.kind)
        assertEquals(R.string.player_skip_credits, button?.labelRes)
        assertEquals(90.0, button?.endSeconds ?: 0.0, 0.0)
        assertNull(activeSegmentButton(90.0, segments))
    }

    @Test
    fun outsideSegmentsReturnsNull() {
        assertNull(activeSegmentButton(0.0, segments))
        assertNull(activeSegmentButton(50.0, segments))
        assertNull(activeSegmentButton(Double.NaN, segments))
        assertNull(activeSegmentButton(10.0, emptyList()))
    }

    @Test
    fun unknownKindsAreIgnored() {
        val unknown = listOf(PlaySegment(kind = "other", start = 10.0, end = 20.0))
        assertNull(activeSegmentButton(10.0, unknown))
    }

    @Test
    fun creditsOfferNextEpisodeWhenThereIsOne() {
        val button = activeSegmentButton(80.0, segments, hasNextEpisode = true)
        assertEquals(SegmentButtonKind.Credits, button?.kind)
        assertEquals(R.string.player_next_episode, button?.labelRes)
        assertEquals(SegmentAction.Next, button?.action)
        assertEquals(90.0, button?.endSeconds ?: 0.0, 0.0)
    }

    @Test
    fun countdownNeedsAutoplayNextEpisodeAndTheEnd() {
        val next = NextEpisode(id = 8, title = "Siguiente", season = 2, episode = 3)
        assertTrue(shouldStartNextEpisodeCountdown(true, next, playbackEnded = true))
        assertFalse(shouldStartNextEpisodeCountdown(false, next, playbackEnded = true))
        assertFalse(shouldStartNextEpisodeCountdown(true, null, playbackEnded = true))
        assertFalse(shouldStartNextEpisodeCountdown(true, next, playbackEnded = false))
    }
}
