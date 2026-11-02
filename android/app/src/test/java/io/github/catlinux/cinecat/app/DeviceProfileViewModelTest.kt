package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.SessionProfile
import java.io.IOException
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class DeviceProfileViewModelTest {
    @Test
    fun loadsTheCurrentProfile() = runTest {
        val profile = SessionProfile(
            maxQuality = "720",
            audioLanguage = "spa",
            subtitleLanguage = "off"
        )
        val viewModel = viewModel(
            loadProfile = { ApiResult.Success(profile) },
            saveProfile = { ApiResult.Success(it) }
        )

        advanceUntilIdle()

        val state = viewModel.state.value as DeviceProfileUiState.Ready
        assertEquals(profile, state.profile)
        assertNull(state.saveResult)
    }

    @Test
    fun hidesTheSectionOnAnOldServer() = runTest {
        val viewModel = viewModel(
            loadProfile = { ApiResult.HttpError(404, "sesión no encontrada") }
        )

        advanceUntilIdle()

        assertEquals(DeviceProfileUiState.Unsupported, viewModel.state.value)
    }

    @Test
    fun reportsLoadErrors() = runTest {
        val viewModel = viewModel(
            loadProfile = { ApiResult.NetworkError(IOException("sin red")) }
        )

        advanceUntilIdle()

        assertTrue(viewModel.state.value is DeviceProfileUiState.Error)
    }

    @Test
    fun cyclesQualityAndSavesTheWholeProfile() = runTest {
        val saved = mutableListOf<SessionProfile>()
        val viewModel = viewModel(
            loadProfile = { ApiResult.Success(SessionProfile()) },
            saveProfile = {
                saved += it
                ApiResult.Success(it)
            }
        )
        advanceUntilIdle()

        viewModel.cycleMaxQuality()
        advanceUntilIdle()

        assertEquals(listOf(SessionProfile(maxQuality = "480")), saved)
        val state = viewModel.state.value as DeviceProfileUiState.Ready
        assertEquals("480", state.profile.maxQuality)
        assertEquals(DeviceProfileSaveResult.Saved, state.saveResult)
        assertTrue(!state.saving)
    }

    @Test
    fun keepsTheNewValueWhenSavingFails() = runTest {
        val viewModel = viewModel(
            loadProfile = { ApiResult.Success(SessionProfile()) },
            saveProfile = { ApiResult.HttpError(500, "error interno") }
        )
        advanceUntilIdle()

        viewModel.cycleSubtitleLanguage()
        advanceUntilIdle()

        val state = viewModel.state.value as DeviceProfileUiState.Ready
        assertEquals("off", state.profile.subtitleLanguage)
        assertEquals(DeviceProfileSaveResult.Failed, state.saveResult)
    }

    @Test
    fun hidesTheSectionIfSavingFindsAnOldServer() = runTest {
        val viewModel = viewModel(
            loadProfile = { ApiResult.Success(SessionProfile()) },
            saveProfile = { ApiResult.HttpError(404, "sesión no encontrada") }
        )
        advanceUntilIdle()

        viewModel.cycleAudioLanguage()
        advanceUntilIdle()

        assertEquals(DeviceProfileUiState.Unsupported, viewModel.state.value)
    }

    @Test
    fun keepsUnknownLanguagesAvailableInTheCycle() = runTest {
        val saved = mutableListOf<SessionProfile>()
        val viewModel = viewModel(
            loadProfile = {
                ApiResult.Success(
                    SessionProfile(audioLanguage = "spa", subtitleLanguage = "zz")
                )
            },
            saveProfile = {
                saved += it
                ApiResult.Success(it)
            }
        )
        advanceUntilIdle()

        viewModel.cycleAudioLanguage()
        advanceUntilIdle()
        assertEquals("", (viewModel.state.value as DeviceProfileUiState.Ready).profile.audioLanguage)

        repeat(PROFILE_AUDIO_LANGUAGES.size) {
            viewModel.cycleAudioLanguage()
            advanceUntilIdle()
        }
        assertEquals("spa", (viewModel.state.value as DeviceProfileUiState.Ready).profile.audioLanguage)

        viewModel.cycleSubtitleLanguage()
        advanceUntilIdle()
        assertEquals(
            "",
            (viewModel.state.value as DeviceProfileUiState.Ready).profile.subtitleLanguage
        )
        repeat(PROFILE_SUBTITLE_LANGUAGES.size) {
            viewModel.cycleSubtitleLanguage()
            advanceUntilIdle()
        }
        assertEquals(
            "zz",
            (viewModel.state.value as DeviceProfileUiState.Ready).profile.subtitleLanguage
        )
        assertTrue(saved.isNotEmpty())
    }

    private fun kotlinx.coroutines.test.TestScope.viewModel(
        loadProfile: suspend () -> ApiResult<SessionProfile> = {
            ApiResult.Success(SessionProfile())
        },
        saveProfile: suspend (SessionProfile) -> ApiResult<SessionProfile> = {
            ApiResult.Success(it)
        }
    ): DeviceProfileViewModel = DeviceProfileViewModel(
        loadProfile = loadProfile,
        saveProfile = saveProfile,
        scope = this
    )
}
