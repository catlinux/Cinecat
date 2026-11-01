package io.github.catlinux.cinecat.app

import java.util.Locale
import org.junit.Assert.assertEquals
import org.junit.Test

class PlayerFormattingTest {
    private val spanish = Locale.forLanguageTag("es-ES")
    private val audioFormat = "Audio %1\$d"
    private val subtitleFormat = "Subtítulo %1\$d"

    @Test
    fun formatsHoursAndMinutes() {
        assertEquals("0:00", formatPlaybackPosition(0.0))
        assertEquals("0:42", formatPlaybackPosition(42.9))
        assertEquals("2:05", formatPlaybackPosition(125.0))
        assertEquals("1:02:15", formatPlaybackPosition(3735.0))
    }

    @Test
    fun prefersTheTrackLabel() {
        assertEquals(
            "Comentario del director",
            formatPlayerTrackLabel(
                label = "  Comentario del director  ",
                language = "en",
                kind = PlayerTrackKind.Audio,
                position = 2,
                audioFormat = audioFormat,
                subtitleFormat = subtitleFormat,
                locale = spanish
            )
        )
    }

    @Test
    fun translatesTheLanguageWhenThereIsNoLabel() {
        assertEquals(
            "Inglés",
            formatPlayerTrackLabel(
                label = " ",
                language = "en",
                kind = PlayerTrackKind.Audio,
                position = 1,
                audioFormat = audioFormat,
                subtitleFormat = subtitleFormat,
                locale = spanish
            )
        )
        assertEquals(
            "Español",
            formatPlayerTrackLabel(
                label = null,
                language = "es",
                kind = PlayerTrackKind.Subtitle,
                position = 2,
                audioFormat = audioFormat,
                subtitleFormat = subtitleFormat,
                locale = spanish
            )
        )
    }

    @Test
    fun numbersTracksWithoutLabelOrLanguage() {
        assertEquals(
            "Audio 2",
            formatPlayerTrackLabel(
                label = null,
                language = "und",
                kind = PlayerTrackKind.Audio,
                position = 2,
                audioFormat = audioFormat,
                subtitleFormat = subtitleFormat,
                locale = spanish
            )
        )
        assertEquals(
            "Subtítulo 1",
            formatPlayerTrackLabel(
                label = "",
                language = null,
                kind = PlayerTrackKind.Subtitle,
                position = 1,
                audioFormat = audioFormat,
                subtitleFormat = subtitleFormat,
                locale = spanish
            )
        )
    }
}
