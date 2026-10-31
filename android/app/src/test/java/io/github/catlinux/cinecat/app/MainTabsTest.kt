package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.SectionsResponse
import org.junit.Assert.assertEquals
import org.junit.Test

class MainTabsTest {
    @Test
    fun hidesMoviesAndCollectionsWithoutAccessibleMovies() {
        val visible = visibleMainTabs(
            mainTabs(),
            SectionsResponse(movies = false, series = true, collections = false)
        )

        assertEquals(listOf(Portal, Series, Search), visible.map { it.route })
    }

    @Test
    fun hidesSeriesWithoutAccessibleSeries() {
        val visible = visibleMainTabs(
            mainTabs(),
            SectionsResponse(movies = true, series = false, collections = true)
        )

        assertEquals(listOf(Portal, Movies, Collections, Search), visible.map { it.route })
    }

    @Test
    fun showsEveryTabWhenSectionsCouldNotBeLoaded() {
        val visible = visibleMainTabs(mainTabs(), sections = null)

        assertEquals(mainTabs().map { it.route }, visible.map { it.route })
    }

    @Test
    fun keepsPortalAndSearchWhenThereIsNoContent() {
        val visible = visibleMainTabs(
            mainTabs(),
            SectionsResponse(movies = false, series = false, collections = false)
        )

        assertEquals(listOf(Portal, Search), visible.map { it.route })
    }
}
