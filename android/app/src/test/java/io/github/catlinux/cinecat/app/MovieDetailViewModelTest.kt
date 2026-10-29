package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.SimilarItemsResponse
import io.github.catlinux.cinecat.core.api.UserList
import java.io.IOException
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class MovieDetailViewModelTest {
    @Test
    fun loadsItemAndSimilarInParallel() = runTest {
        val item = sampleMediaItem(1)
        val similar = sampleMediaItem(2)
        val viewModel = viewModel(
            loadItem = { ApiResult.Success(item) },
            loadSimilar = { ApiResult.Success(SimilarItemsResponse(listOf(similar))) }
        )

        advanceUntilIdle()

        val ready = viewModel.state.value as MovieDetailUiState.Ready
        assertEquals(item, ready.item)
        assertEquals(listOf(similar), ready.similar)
    }

    @Test
    fun similarFailureDoesNotBreakTheItem() = runTest {
        val item = sampleMediaItem(1)
        val viewModel = viewModel(
            loadItem = { ApiResult.Success(item) },
            loadSimilar = { ApiResult.NetworkError(IOException("sin red")) }
        )

        advanceUntilIdle()

        val ready = viewModel.state.value as MovieDetailUiState.Ready
        assertEquals(item, ready.item)
        assertTrue(ready.similar.isEmpty())
    }

    @Test
    fun itemFailureCanBeRetried() = runTest {
        var calls = 0
        val item = sampleMediaItem(1)
        val viewModel = viewModel(
            loadItem = {
                calls++
                if (calls == 1) {
                    ApiResult.NetworkError(IOException("sin red"))
                } else {
                    ApiResult.Success(item)
                }
            },
            loadSimilar = { ApiResult.Success(SimilarItemsResponse(emptyList())) }
        )

        advanceUntilIdle()
        assertEquals(MovieDetailUiState.Error, viewModel.state.value)

        viewModel.retry()
        advanceUntilIdle()

        assertTrue(viewModel.state.value is MovieDetailUiState.Ready)
    }

    @Test
    fun favoriteIsOptimisticAndUsesSetThenClear() = runTest {
        var setCalls = 0
        var clearCalls = 0
        val viewModel = viewModel(
            loadItem = { ApiResult.Success(sampleMediaItem(1)) },
            loadSimilar = { ApiResult.Success(SimilarItemsResponse(emptyList())) },
            setItemList = { list ->
                if (list == UserList.Favorite) setCalls++
                ApiResult.Success(Unit)
            },
            clearItemList = { list ->
                if (list == UserList.Favorite) clearCalls++
                ApiResult.Success(Unit)
            }
        )
        advanceUntilIdle()

        viewModel.toggleFavorite()

        assertTrue((viewModel.state.value as MovieDetailUiState.Ready).item.favorite)
        advanceUntilIdle()
        assertEquals(1, setCalls)

        viewModel.toggleFavorite()
        advanceUntilIdle()

        assertFalse((viewModel.state.value as MovieDetailUiState.Ready).item.favorite)
        assertEquals(1, clearCalls)
    }

    @Test
    fun watchedUsesPutAndDelete() = runTest {
        var setCalls = 0
        var clearCalls = 0
        val viewModel = viewModel(
            loadItem = { ApiResult.Success(sampleMediaItem(1)) },
            loadSimilar = { ApiResult.Success(SimilarItemsResponse(emptyList())) },
            setWatched = {
                setCalls++
                ApiResult.Success(Unit)
            },
            clearWatched = {
                clearCalls++
                ApiResult.Success(Unit)
            }
        )
        advanceUntilIdle()

        viewModel.toggleWatched()
        advanceUntilIdle()
        viewModel.toggleWatched()
        advanceUntilIdle()

        assertEquals(1, setCalls)
        assertEquals(1, clearCalls)
    }

    @Test
    fun failedActionRollsBackAndEmitsOneError() = runTest {
        val events = mutableListOf<DetailAction>()
        val viewModel = viewModel(
            loadItem = { ApiResult.Success(sampleMediaItem(1)) },
            loadSimilar = { ApiResult.Success(SimilarItemsResponse(emptyList())) },
            setItemList = { ApiResult.NetworkError(IOException("sin red")) }
        )
        advanceUntilIdle()

        viewModel.toggleFavorite()
        advanceUntilIdle()

        assertFalse((viewModel.state.value as MovieDetailUiState.Ready).item.favorite)
        events += viewModel.events.first()
        assertEquals(listOf(DetailAction.Favorite), events)
    }

    @Test
    fun repeatedActionWhilePendingIsIgnored() = runTest {
        var calls = 0
        val viewModel = viewModel(
            loadItem = { ApiResult.Success(sampleMediaItem(1)) },
            loadSimilar = { ApiResult.Success(SimilarItemsResponse(emptyList())) },
            setItemList = {
                calls++
                delay(1_000)
                ApiResult.Success(Unit)
            }
        )
        advanceUntilIdle()

        viewModel.toggleFavorite()
        viewModel.toggleFavorite()
        advanceUntilIdle()

        assertEquals(1, calls)
        assertTrue((viewModel.state.value as MovieDetailUiState.Ready).item.favorite)
    }

    private fun kotlinx.coroutines.test.TestScope.viewModel(
        loadItem: suspend () -> ApiResult<io.github.catlinux.cinecat.core.api.MediaItem>,
        loadSimilar: suspend () -> ApiResult<SimilarItemsResponse>,
        setItemList: suspend (UserList) -> ApiResult<Unit> = { ApiResult.Success(Unit) },
        clearItemList: suspend (UserList) -> ApiResult<Unit> = { ApiResult.Success(Unit) },
        setWatched: suspend () -> ApiResult<Unit> = { ApiResult.Success(Unit) },
        clearWatched: suspend () -> ApiResult<Unit> = { ApiResult.Success(Unit) }
    ): MovieDetailViewModel = MovieDetailViewModel(
        id = 1,
        loadItem = loadItem,
        loadSimilar = loadSimilar,
        setItemList = setItemList,
        clearItemList = clearItemList,
        setWatched = setWatched,
        clearWatched = clearWatched,
        scope = this
    )
}
