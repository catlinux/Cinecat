package io.github.catlinux.cinecat.app

import org.junit.Assert.assertEquals
import org.junit.Test

class PlayerBurnSubtitlesTest {
    @Test
    fun hlsKeepsEveryBurnSubtitle() {
        val burn = listOf(
            burn("e0", "es"),
            burn("e1", "spa")
        )

        assertEquals(burn, visibleBurnSubtitles(burn, listOf("es", "spa"), isHls = true))
    }

    @Test
    fun directHidesTheSameLanguageInBothDirections() {
        assertEquals(
            emptyList<PlayerBurnSubtitle>(),
            visibleBurnSubtitles(listOf(burn("e0", "spa")), listOf("es"), isHls = false)
        )
        assertEquals(
            emptyList<PlayerBurnSubtitle>(),
            visibleBurnSubtitles(listOf(burn("e0", "es")), listOf("spa"), isHls = false)
        )
    }

    @Test
    fun directKeepsEntriesWithoutANativeMatch() {
        val burn = listOf(
            burn("e0", "es"),
            burn("e1", null)
        )

        assertEquals(burn, visibleBurnSubtitles(burn, listOf("fr"), isHls = false))
    }

    @Test
    fun oneNativeImageOnlyHidesOneOfTwoMatchingEntries() {
        val burn = listOf(
            burn("e0", "es"),
            burn("e1", "es")
        )

        assertEquals(
            listOf("e1"),
            visibleBurnSubtitles(burn, listOf("spa"), isHls = false).map { it.id }
        )
    }

    @Test
    fun entriesWithoutLanguageOnlyMatchOtherEntriesWithoutLanguage() {
        val burn = listOf(
            burn("e0", null),
            burn("e1", ""),
            burn("e2", "es")
        )

        assertEquals(
            listOf("e2"),
            visibleBurnSubtitles(burn, listOf(null, ""), isHls = false).map { it.id }
        )
        assertEquals(
            listOf("e0", "e1", "e2"),
            visibleBurnSubtitles(burn, listOf("fr"), isHls = false).map { it.id }
        )
    }

    private fun burn(id: String, language: String?): PlayerBurnSubtitle = PlayerBurnSubtitle(
        id = id,
        label = id,
        selected = false,
        language = language
    )
}
