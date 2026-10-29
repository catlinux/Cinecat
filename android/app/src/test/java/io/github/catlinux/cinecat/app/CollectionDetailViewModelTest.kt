package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.UserList
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class CollectionDetailViewModelTest {
    @Test
    fun loadsAndTogglesCollectionLists() = runTest {
        var favoriteSet = 0
        var watchlistSet = 0
        val collection = sampleCollectionDetail()
        val viewModel = CollectionDetailViewModel(
            id = collection.id,
            loadCollection = { ApiResult.Success(collection) },
            setCollectionList = { list ->
                when (list) {
                    UserList.Favorite -> favoriteSet++
                    UserList.Watchlist -> watchlistSet++
                }
                ApiResult.Success(Unit)
            },
            clearCollectionList = { ApiResult.Success(Unit) },
            scope = this
        )

        advanceUntilIdle()
        assertTrue(viewModel.state.value is CollectionDetailUiState.Ready)

        viewModel.toggleFavorite()
        advanceUntilIdle()
        viewModel.toggleWatchlist()
        advanceUntilIdle()

        val ready = viewModel.state.value as CollectionDetailUiState.Ready
        assertTrue(ready.collection.favorite)
        assertTrue(ready.collection.watchlist)
        assertEquals(1, favoriteSet)
        assertEquals(1, watchlistSet)
    }
}
