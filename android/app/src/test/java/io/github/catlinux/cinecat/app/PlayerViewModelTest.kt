package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.AudioTrack
import io.github.catlinux.cinecat.core.api.ItemPreferences
import io.github.catlinux.cinecat.core.api.PlayResponse
import io.github.catlinux.cinecat.core.api.PreferencesPatch
import io.github.catlinux.cinecat.core.api.QualityOption
import io.github.catlinux.cinecat.core.api.SubtitleOption
import java.io.IOException
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class PlayerViewModelTest {
    @Test
    fun resolvesRelativeUrlsAndDetectsHls() = runTest {
        val origin = "http://servidor.local:8096".toHttpUrl()
        val viewModel = viewModel(
            origin = origin,
            play = {
                ApiResult.Success(
                    PlayResponse(
                        url = "/api/v1/stream/7?exp=1&sig=abc",
                        mode = "remux",
                        subtitles = listOf(
                            SubtitleOption(
                                id = "e0",
                                language = "es",
                                languageName = "Español",
                                title = "Español",
                                url = "/api/v1/stream/7/subtitles/e0.vtt"
                            )
                        ),
                        selectedSubtitle = "e0"
                    )
                )
            }
        )

        advanceUntilIdle()

        val ready = viewModel.state.value as PlayerUiState.Ready
        assertEquals(
            "http://servidor.local:8096/api/v1/stream/7?exp=1&sig=abc",
            ready.streamUrl
        )
        assertTrue(ready.isHls)
        assertEquals(
            "http://servidor.local:8096/api/v1/stream/7/subtitles/e0.vtt",
            ready.subtitles.single().url
        )
        assertEquals("es", ready.subtitles.single().language)
        assertEquals("Español", ready.subtitles.single().label)
        assertTrue(ready.subtitles.single().selected)
    }

    @Test
    fun directErrorFallsBackToHlsOnlyOnceAndKeepsPosition() = runTest {
        val origin = "http://servidor.local:8096".toHttpUrl()
        val calls = mutableListOf<String?>()
        val viewModel = viewModel(
            origin = origin,
            play = { caps ->
                calls += caps
                if (caps == null) {
                    ApiResult.Success(
                        PlayResponse(
                            url = "/api/v1/stream/7/playlist.m3u8",
                            mode = "remux"
                        )
                    )
                } else {
                    ApiResult.Success(
                        PlayResponse(
                            url = "/api/v1/stream/7?exp=1&sig=abc",
                            mode = "direct"
                        )
                    )
                }
            }
        )
        advanceUntilIdle()
        assertEquals(listOf("caps"), calls)

        viewModel.onPlaybackError(42.5)
        advanceUntilIdle()

        assertEquals(listOf("caps", null), calls)
        val ready = viewModel.state.value as PlayerUiState.Ready
        assertTrue(ready.isHls)
        assertEquals(42.5, ready.initialPositionSeconds, 0.0)

        viewModel.onPlaybackError(43.0)
        advanceUntilIdle()
        assertTrue(viewModel.state.value is PlayerUiState.Error)
        assertEquals(2, calls.size)
    }

    @Test
    fun failedPlaybackCanBeRetried() = runTest {
        var attempts = 0
        val viewModel = viewModel(
            play = {
                attempts++
                if (attempts == 1) {
                    ApiResult.NetworkError(IOException("sin red"))
                } else {
                    ApiResult.Success(
                        PlayResponse(
                            url = "/api/v1/stream/7?exp=1&sig=abc",
                            mode = "direct"
                        )
                    )
                }
            }
        )

        advanceUntilIdle()
        assertTrue(viewModel.state.value is PlayerUiState.Error)

        viewModel.retry()
        advanceUntilIdle()

        assertTrue(viewModel.state.value is PlayerUiState.Ready)
        assertEquals(2, attempts)
    }

    @Test
    fun saveProgressSkipsUnknownDuration() = runTest {
        val saved = mutableListOf<Pair<Double, Double>>()
        val viewModel = viewModel(
            saveProgress = { position, duration ->
                saved += position to duration
                ApiResult.Success(Unit)
            }
        )
        advanceUntilIdle()

        viewModel.saveProgress(12.5, 0.0)
        advanceUntilIdle()
        assertTrue(saved.isEmpty())

        viewModel.saveProgress(12.5, 100.0)
        advanceUntilIdle()
        assertEquals(listOf(12.5 to 100.0), saved)
    }

    @Test
    fun usesSavedPreferencesThatStillExist() = runTest {
        val preferences = ItemPreferences(
            audioIndex = 1,
            subtitleId = "e1",
            quality = "720"
        )
        val viewModel = viewModel(
            loadPreferences = { ApiResult.Success(preferences) },
            play = {
                ApiResult.Success(
                    PlayResponse(
                        url = "/stream",
                        mode = "remux",
                        audioTracks = listOf(
                            AudioTrack(index = 0, language = "es"),
                            AudioTrack(index = 1, language = "en")
                        ),
                        selectedAudio = 0,
                        qualities = listOf(
                            QualityOption("auto", "Auto"),
                            QualityOption("720", "720p")
                        ),
                        selectedQuality = "auto",
                        subtitles = listOf(
                            SubtitleOption(id = "e0", language = "es", url = "/e0.vtt"),
                            SubtitleOption(id = "e1", language = "en", url = "/e1.vtt")
                        ),
                        selectedSubtitle = "e0"
                    )
                )
            }
        )
        advanceUntilIdle()

        val ready = viewModel.state.value as PlayerUiState.Ready
        assertEquals(1, ready.selectedAudio)
        assertEquals("e1", ready.selectedSubtitle)
        assertEquals("720", ready.selectedQuality)
        assertEquals(listOf(false, true), ready.subtitles.map { it.selected })
    }

    @Test
    fun preferenceFailuresDoNotStopPlaybackOrSavingOtherChanges() = runTest {
        val saved = mutableListOf<PreferencesPatch>()
        val viewModel = viewModel(
            loadPreferences = { ApiResult.NetworkError(IOException("sin red")) },
            savePreferences = {
                saved += it
                ApiResult.NetworkError(IOException("sin red"))
            }
        )
        advanceUntilIdle()

        assertTrue(viewModel.state.value is PlayerUiState.Ready)
        viewModel.savePlaybackPreferences(audioIndex = 1, quality = "720")
        advanceUntilIdle()

        assertEquals(
            listOf(PreferencesPatch(audioIndex = 1, quality = "720")),
            saved
        )
        assertTrue(viewModel.state.value is PlayerUiState.Ready)
    }

    private fun TestScope.viewModel(
        origin: okhttp3.HttpUrl? = "http://servidor.local".toHttpUrl(),
        play: suspend (String?) -> ApiResult<PlayResponse> = {
            ApiResult.Success(PlayResponse(url = "/stream", mode = "direct"))
        },
        saveProgress: suspend (Double, Double) -> ApiResult<Unit> = { _, _ ->
            ApiResult.Success(Unit)
        },
        loadPreferences: suspend () -> ApiResult<ItemPreferences> = {
            ApiResult.Success(ItemPreferences())
        },
        savePreferences: suspend (PreferencesPatch) -> ApiResult<Unit> = {
            ApiResult.Success(Unit)
        }
    ): PlayerViewModel = PlayerViewModel(
        itemId = 7,
        startSeconds = 0.0,
        caps = "caps",
        loadPlayback = play,
        persistProgress = saveProgress,
        loadPreferences = loadPreferences,
        persistPreferences = savePreferences,
        serverUrl = { origin },
        scope = this
    )
}
