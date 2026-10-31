package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.AudioTrack
import io.github.catlinux.cinecat.core.api.ItemPreferences
import io.github.catlinux.cinecat.core.api.PlayResponse
import io.github.catlinux.cinecat.core.api.QualityOption
import io.github.catlinux.cinecat.core.api.SubtitleOption
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PlayerPreferencesTest {
    private fun response() = PlayResponse(
        url = "/stream",
        mode = "remux",
        audioTracks = listOf(
            AudioTrack(index = 0, language = "es"),
            AudioTrack(index = 1, language = "en")
        ),
        selectedAudio = 0,
        qualities = listOf(
            QualityOption("auto", "Auto"),
            QualityOption("720", "720p")
        ),
        selectedQuality = "auto",
        subtitles = listOf(
            SubtitleOption(id = "e0", language = "es"),
            SubtitleOption(id = "e1", language = "en")
        ),
        selectedSubtitle = "e0"
    )

    @Test
    fun appliesSavedPreferencesThatStillExist() {
        val resolved = resolvePlayerPreferences(
            response(),
            ItemPreferences(audioIndex = 1, subtitleId = "e1", quality = "720")
        )
        assertEquals(1, resolved.audioIndex)
        assertEquals("e1", resolved.subtitleId)
        assertEquals("720", resolved.quality)
    }

    @Test
    fun ignoresPreferencesThatNoLongerExist() {
        val resolved = resolvePlayerPreferences(
            response(),
            ItemPreferences(audioIndex = 9, subtitleId = "e9", quality = "480")
        )
        assertEquals(0, resolved.audioIndex)
        assertEquals("e0", resolved.subtitleId)
        assertEquals("auto", resolved.quality)
    }

    @Test
    fun emptySubtitlePreferenceMeansNoSubtitles() {
        val resolved = resolvePlayerPreferences(
            response(),
            ItemPreferences(subtitleId = "")
        )
        assertNull(resolved.subtitleId)
    }

    @Test
    fun withoutPreferencesKeepsTheServerSelection() {
        val resolved = resolvePlayerPreferences(response(), null)
        assertEquals(0, resolved.audioIndex)
        assertEquals("e0", resolved.subtitleId)
        assertEquals("auto", resolved.quality)
    }
}
