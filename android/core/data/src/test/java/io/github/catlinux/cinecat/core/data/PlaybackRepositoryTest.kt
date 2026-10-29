package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.createApi
import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class PlaybackRepositoryTest {
    @get:Rule
    val server = MockWebServer()

    @Test
    fun playPassesCapabilitiesAndParsesTheResponse() = runTest {
        val api = createApi(server.url("/"), OkHttpClient())
        val repository = PlaybackRepository(apiProvider = { api })
        server.enqueue(
            jsonResponse(
                """{
                  "url":"/api/v1/stream/7?exp=1&sig=abc",
                  "mode":"direct",
                  "expires_at":"2026-10-30T00:00:00Z",
                  "audio_tracks":[],
                  "selected_audio":0,
                  "qualities":[],
                  "selected_quality":"auto",
                  "subtitles":[],
                  "selected_subtitle":null
                }"""
            )
        )

        val result = repository.play(
            id = 7,
            caps = "mkv,mp4,h264",
            audio = 0,
            quality = "auto",
            subtitle = ""
        )

        assertTrue(result is ApiResult.Success)
        assertEquals("direct", (result as ApiResult.Success).value.mode)
        val request = server.takeRequest()
        assertEquals("/api/v1/items/7/play", request.requestUrl?.encodedPath)
        assertEquals("mkv,mp4,h264", request.requestUrl?.queryParameter("caps"))
        assertEquals("0", request.requestUrl?.queryParameter("audio"))
        assertEquals("auto", request.requestUrl?.queryParameter("quality"))
        assertEquals("", request.requestUrl?.queryParameter("subtitle"))
    }

    @Test
    fun saveProgressSendsTheProgressRequest() = runTest {
        val api = createApi(server.url("/"), OkHttpClient())
        val repository = PlaybackRepository(apiProvider = { api })
        server.enqueue(MockResponse().setResponseCode(204))

        assertTrue(repository.saveProgress(7, 12.5, 100.0) is ApiResult.Success)
        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/items/7/progress", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"position_seconds\":12.5"))
        assertTrue(body.contains("\"duration_seconds\":100.0"))
    }

    @Test
    fun withoutServerReturnsNetworkError() = runTest {
        val repository = PlaybackRepository(apiProvider = { null })
        assertTrue(repository.play(7) is ApiResult.NetworkError)
        assertTrue(repository.saveProgress(7, 1.0, 2.0) is ApiResult.NetworkError)
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)
}
