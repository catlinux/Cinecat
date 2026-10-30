package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.LetterIndex
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TvLetterIndexTest {
    private val letters = listOf(
        LetterIndex("#", 0, 2),
        LetterIndex("A", 2, 3),
        LetterIndex("B", 5, 1)
    )

    @Test
    fun returnsTheLetterThatOwnsTheVisibleIndex() {
        assertEquals("#", letterForIndex(letters, 0))
        assertEquals("#", letterForIndex(letters, 1))
        assertEquals("A", letterForIndex(letters, 2))
        assertEquals("A", letterForIndex(letters, 4))
        assertEquals("B", letterForIndex(letters, 5))
    }

    @Test
    fun returnsNullOutsideTheIndexedRange() {
        assertNull(letterForIndex(letters, -1))
        assertNull(letterForIndex(letters, 6))
        assertNull(letterForIndex(emptyList(), 0))
    }
}
