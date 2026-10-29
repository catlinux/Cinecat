package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.data.PortalData
import io.github.catlinux.cinecat.core.data.PortalRowData
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class PortalViewModelTest {
    @Test
    fun loadsReadyPortal() = runTest {
        val data = portalData()
        val viewModel = PortalViewModel(
            loadPortal = { ApiResult.Success(data) },
            scope = this
        )

        advanceUntilIdle()

        val state = viewModel.state.value
        assertTrue(state is PortalUiState.Ready)
        assertEquals(data, (state as PortalUiState.Ready).data)
    }

    @Test
    fun reportsErrorAndRetries() = runTest {
        var calls = 0
        val viewModel = PortalViewModel(
            loadPortal = {
                calls++
                if (calls == 1) {
                    ApiResult.HttpError(500, "boom")
                } else {
                    ApiResult.Success(portalData())
                }
            },
            scope = this
        )

        advanceUntilIdle()
        assertTrue(viewModel.state.value is PortalUiState.Error)

        viewModel.retry()
        advanceUntilIdle()
        assertTrue(viewModel.state.value is PortalUiState.Ready)
        assertEquals(2, calls)
    }

    @Test
    fun emptyPortalHasItsOwnState() = runTest {
        val viewModel = PortalViewModel(
            loadPortal = { ApiResult.Success(PortalData(emptyList())) },
            scope = this
        )

        advanceUntilIdle()

        assertEquals(PortalUiState.Empty, viewModel.state.value)
    }

    private fun portalData(): PortalData = PortalData(
        rows = listOf(
            PortalRowData(
                id = "recent_movies",
                title = "Recientes",
                kicker = "Estrenos",
                entries = emptyList()
            )
        )
    )
}
