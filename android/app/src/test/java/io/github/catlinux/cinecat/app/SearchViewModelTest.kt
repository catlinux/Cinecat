package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.ItemsResponse
import io.github.catlinux.cinecat.core.api.SeriesResponse
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.delay
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class SearchViewModelTest {
    @Test
    fun waitsThreeHundredMillisecondsAndNeedsTwoCharacters() = runTest {
        val queries = mutableListOf<String>()
        val viewModel = SearchViewModel(
            loadMovies = { query ->
                queries += query
                ApiResult.Success(ItemsResponse(emptyList(), 0, 30, 0))
            },
            loadSeries = {
                ApiResult.Success(SeriesResponse(emptyList(), 0, 30, 0))
            },
            scope = this
        )

        viewModel.onQueryChanged("a")
        advanceUntilIdle()
        assertTrue(queries.isEmpty())

        viewModel.onQueryChanged("ab")
        advanceTimeBy(299)
        runCurrent()
        assertTrue(queries.isEmpty())

        advanceTimeBy(1)
        advanceUntilIdle()
        assertEquals(listOf("ab"), queries)
    }

    @Test
    fun aNewQueryCancelsThePreviousOne() = runTest {
        val queries = mutableListOf<String>()
        val viewModel = SearchViewModel(
            loadMovies = { query ->
                queries += query
                delay(1_000)
                ApiResult.Success(ItemsResponse(emptyList(), 0, 30, 0))
            },
            loadSeries = {
                delay(1_000)
                ApiResult.Success(SeriesResponse(emptyList(), 0, 30, 0))
            },
            scope = this
        )

        viewModel.onQueryChanged("ab")
        advanceTimeBy(300)
        runCurrent()
        assertEquals(listOf("ab"), queries)

        viewModel.onQueryChanged("abc")
        advanceTimeBy(300)
        runCurrent()
        advanceUntilIdle()

        assertEquals(listOf("ab", "abc"), queries)
        assertEquals("abc", viewModel.state.value.query)
        assertTrue(viewModel.state.value.hasSearched)
    }
}
