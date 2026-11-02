package io.github.catlinux.cinecat.app

import org.junit.Assert.assertEquals
import org.junit.Test

class DeviceProfileOptionsTest {
    @Test
    fun cyclesMaxQualityAndWraps() {
        assertEquals("480", nextProfileValue("", PROFILE_MAX_QUALITIES))
        assertEquals("720", nextProfileValue("480", PROFILE_MAX_QUALITIES))
        assertEquals("1080", nextProfileValue("720", PROFILE_MAX_QUALITIES))
        assertEquals("", nextProfileValue("1080", PROFILE_MAX_QUALITIES))
    }

    @Test
    fun cyclesAudioAndSubtitleLanguages() {
        assertEquals("es", nextProfileValue("", PROFILE_AUDIO_LANGUAGES))
        assertEquals("", nextProfileValue("ja", PROFILE_AUDIO_LANGUAGES))
        assertEquals("off", nextProfileValue("", PROFILE_SUBTITLE_LANGUAGES))
        assertEquals("es", nextProfileValue("off", PROFILE_SUBTITLE_LANGUAGES))
        assertEquals("", nextProfileValue("ja", PROFILE_SUBTITLE_LANGUAGES))
    }

    @Test
    fun keepsUnknownLanguagesInTheCycle() {
        val audioOptions = PROFILE_AUDIO_LANGUAGES + "spa"
        assertEquals("spa", nextProfileValue("ja", audioOptions))
        assertEquals("", nextProfileValue("spa", audioOptions))
        val subtitleOptions = PROFILE_SUBTITLE_LANGUAGES + "zz"
        assertEquals("zz", nextProfileValue("ja", subtitleOptions))
        assertEquals("", nextProfileValue("zz", subtitleOptions))
    }
}
