package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.SectionsResponse
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
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
    fun showsEverySectionWhenSectionsCouldNotBeLoadedButOnlyShowsPhotosWithLibraries() {
        val withoutPhotos = visibleMainTabs(mainTabs(), sections = null)
        assertEquals(mainTabs().filter { it.route != Photos }.map { it.route }, withoutPhotos.map { it.route })

        val withPhotos = visibleMainTabs(mainTabs(), sections = null, hasPhotos = true)
        assertEquals(mainTabs().map { it.route }, withPhotos.map { it.route })
    }

    @Test
    fun keepsPortalAndSearchWhenThereIsNoContent() {
        val visible = visibleMainTabs(
            mainTabs(),
            SectionsResponse(movies = false, series = false, collections = false)
        )

        assertEquals(listOf(Portal, Search), visible.map { it.route })
    }

    @Test
    fun photosTabOnlyAppearsWithAccessiblePhotoLibraries() {
        val withoutPhotos = visibleMainTabs(
            mainTabs(),
            SectionsResponse(movies = true, series = true, collections = true),
            hasPhotos = false
        )
        assertTrue(withoutPhotos.none { it.route == Photos })

        val withPhotos = visibleMainTabs(
            mainTabs(),
            SectionsResponse(movies = true, series = true, collections = true),
            hasPhotos = true
        )
        assertTrue(withPhotos.any { it.route == Photos })
    }
}
