package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.GenreCount
import io.github.catlinux.cinecat.core.api.GenresResponse
import io.github.catlinux.cinecat.core.api.ItemsResponse
import io.github.catlinux.cinecat.core.api.MediaItem
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class MoviesViewModelTest {
    private data class Request(val sort: String, val genre: String?, val offset: Int)

    @Test
    fun changingSortAndGenreResetsAndSendsTheFilters() = runTest {
        val requests = mutableListOf<Request>()
        val viewModel = MoviesViewModel(
            loadPage = { sort, genre, offset ->
                requests += Request(sort, genre, offset)
                ApiResult.Success(ItemsResponse(emptyList(), 0, 60, offset))
            },
            loadGenres = {
                ApiResult.Success(GenresResponse(listOf(GenreCount("Drama", 2))))
            },
            scope = this
        )

        advanceUntilIdle()

        assertEquals("title", viewModel.state.value.sort)
        assertEquals(Request("title", null, 0), requests.single())

        viewModel.setSort("year")
        advanceUntilIdle()
        assertEquals("year", viewModel.state.value.sort)
        assertEquals(Request("year", null, 0), requests.last())

        viewModel.setGenre("Drama")
        advanceUntilIdle()
        assertEquals("Drama", viewModel.state.value.genre)
        assertEquals(Request("year", "Drama", 0), requests.last())
        assertTrue(viewModel.state.value.genres.contains("Drama"))
    }

    @Test
    fun loadMoreRequestsTheNextOffset() = runTest {
        val offsets = mutableListOf<Int>()
        val viewModel = MoviesViewModel(
            loadPage = { _, _, offset ->
                offsets += offset
                ApiResult.Success(
                    ItemsResponse(
                        items = List(60) { index -> mediaItem(offset + index.toLong()) },
                        total = 120,
                        limit = 60,
                        offset = offset
                    )
                )
            },
            loadGenres = { ApiResult.Success(GenresResponse(emptyList())) },
            scope = this
        )

        advanceUntilIdle()
        viewModel.loadMore()
        advanceUntilIdle()

        assertEquals(listOf(0, 60), offsets)
    }

    private fun mediaItem(id: Long): MediaItem = MediaItem(
        id = id,
        libraryId = 1,
        libraryKind = "video",
        title = "Pelicula $id",
        fileTitle = "Pelicula $id",
        year = 2024,
        kind = "movie",
        season = null,
        episode = null,
        tmdbId = null,
        overview = null,
        posterUrl = null,
        originalTitle = null,
        tagline = null,
        genres = emptyList(),
        runtimeMinutes = null,
        rating = null,
        directors = emptyList(),
        cast = emptyList(),
        credits = io.github.catlinux.cinecat.core.api.Credits(emptyList(), emptyList()),
        backdropUrl = null,
        watched = false,
        favorite = false,
        watchlist = false,
        progress = null,
        collections = emptyList(),
        seriesId = null,
        seriesTitle = null,
        episodeTitle = null,
        stillUrl = null,
        seriesPosterUrl = null,
        seriesBackdropUrl = null
    )
}
