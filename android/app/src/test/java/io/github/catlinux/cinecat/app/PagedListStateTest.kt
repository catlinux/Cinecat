package io.github.catlinux.cinecat.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PagedListStateTest {
    @Test
    fun firstPageStartsAtZeroAndKnowsThereIsMore() {
        val state = PagedListState<String>()
        val started = state.begin()

        assertEquals(true, started?.loading)
        assertEquals(0, started?.nextOffset)
        val loaded = started!!.success(
            offset = 0,
            total = 125,
            page = List(PagedListState.PAGE_SIZE) { "item-$it" }
        )
        assertEquals(PagedListState.PAGE_SIZE, loaded.items.size)
        assertEquals(PagedListState.PAGE_SIZE, loaded.nextOffset)
        assertTrue(loaded.hasMore)
        assertFalse(loaded.endReached)
    }

    @Test
    fun finishesWhenTotalIsReached() {
        val loaded = PagedListState<String>()
            .begin()!!
            .success(offset = 0, total = 60, page = List(60) { "item-$it" })

        assertFalse(loaded.hasMore)
        assertTrue(loaded.endReached)
        assertNull(loaded.begin())
    }

    @Test
    fun secondPageAppendsAndFinishes() {
        val first = PagedListState<String>()
            .begin()!!
            .success(offset = 0, total = 125, page = List(60) { "first-$it" })
        val second = first.begin()!!
            .success(offset = 60, total = 125, page = List(65) { "second-$it" })

        assertEquals(125, second.items.size)
        assertEquals(125, second.nextOffset)
        assertFalse(second.hasMore)
        assertTrue(second.endReached)
    }

    @Test
    fun simultaneousBeginDoesNotStartTwice() {
        val loading = PagedListState<String>().begin()

        assertTrue(loading != null)
        assertNull(loading!!.begin())
    }

    @Test
    fun resetStartsFromScratch() {
        val loaded = PagedListState<String>()
            .begin()!!
            .success(offset = 0, total = 10, page = List(10) { "item-$it" })

        val reset = loaded.reset()

        assertTrue(reset.items.isEmpty())
        assertEquals(0, reset.nextOffset)
        assertFalse(reset.loading)
        assertFalse(reset.error)
        assertFalse(reset.endReached)
    }

    @Test
    fun aLaterFailureKeepsLoadedItemsAndCanRetry() {
        val loaded = PagedListState<String>()
            .begin()!!
            .success(offset = 0, total = 120, page = List(60) { "item-$it" })
        val failed = loaded.begin()!!.failure()

        assertEquals(60, failed.items.size)
        assertTrue(failed.error)
        assertFalse(failed.loading)
        assertEquals(60, failed.nextOffset)

        val retried = failed.begin()
        assertEquals(60, retried?.nextOffset)
        assertFalse(retried?.error ?: true)
        assertTrue(retried?.loading ?: false)
    }
}
