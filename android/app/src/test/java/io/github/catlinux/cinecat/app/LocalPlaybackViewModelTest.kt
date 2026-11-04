package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.PlayResponse
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LocalPlaybackViewModelTest {
    @Test
    fun laReproduccionLocalNoPidePlayYSalvaElProgreso() = runTest {
        var playbackCalls = 0
        var savedPosition: Double? = null
        var savedDuration: Double? = null
        val viewModel = PlayerViewModel(
            itemId = 7,
            startSeconds = 33.0,
            caps = "caps",
            loadPlayback = {
                playbackCalls++
                ApiResult.Success(PlayResponse(url = "/stream", mode = "direct"))
            },
            persistProgress = { position, duration ->
                savedPosition = position
                savedDuration = duration
                ApiResult.Success(Unit)
            },
            serverUrl = { null },
            scope = this,
            localUri = "https://cine.example/api/v1/stream/7/playlist.m3u8",
            localSubtitles = listOf(PlayerSubtitle("file:///subs/es.vtt", "es", "Español", false, "es"))
        )

        advanceUntilIdle()

        val ready = viewModel.state.value as PlayerUiState.Ready
        assertEquals("https://cine.example/api/v1/stream/7/playlist.m3u8", ready.streamUrl)
        assertEquals(33.0, ready.initialPositionSeconds, 0.0)
        assertTrue(ready.isHls)
        assertEquals(listOf("es"), ready.subtitles.map { it.id })
        assertNull(ready.nextEpisode)
        assertTrue(ready.autoplayNext)
        assertEquals(0, playbackCalls)

        viewModel.saveProgress(40.0, 100.0)
        advanceUntilIdle()

        assertEquals(40.0, savedPosition ?: 0.0, 0.0)
        assertEquals(100.0, savedDuration ?: 0.0, 0.0)
    }
}
