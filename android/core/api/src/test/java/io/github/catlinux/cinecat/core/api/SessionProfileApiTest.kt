package io.github.catlinux.cinecat.core.api

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class SessionProfileApiTest {
    @get:Rule
    val server = MockWebServer()

    private fun api(): CinecatApi = createApi(server.url("/"), OkHttpClient())

    @Test
    fun readsTheCurrentProfileAndUsesDefaultsForMissingFields() = runTest {
        server.enqueue(
            jsonResponse(
                """{"max_quality":"720","audio_language":"spa","subtitle_language":"off"}"""
            )
        )
        val full = api().currentSessionProfile()
        assertTrue(full is ApiResult.Success)
        val value = (full as ApiResult.Success).value
        assertEquals("720", value.maxQuality)
        assertEquals("spa", value.audioLanguage)
        assertEquals("off", value.subtitleLanguage)
        var request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/auth/sessions/current/profile", request.path)

        server.enqueue(jsonResponse("{}"))
        val empty = api().currentSessionProfile()
        assertTrue(empty is ApiResult.Success)
        assertEquals(SessionProfile(), (empty as ApiResult.Success).value)
        request = server.takeRequest()
        assertEquals("/api/v1/auth/sessions/current/profile", request.path)
    }

    @Test
    fun savesTheWholeProfile() = runTest {
        val profile = SessionProfile(
            maxQuality = "1080",
            audioLanguage = "eng",
            subtitleLanguage = "es"
        )
        server.enqueue(
            jsonResponse(
                """{"max_quality":"1080","audio_language":"eng","subtitle_language":"es"}"""
            )
        )
        val result = api().saveCurrentSessionProfile(profile)
        assertTrue(result is ApiResult.Success)
        assertEquals(profile, (result as ApiResult.Success).value)

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/auth/sessions/current/profile", request.path)
        assertEquals(
            """{"max_quality":"1080","audio_language":"eng","subtitle_language":"es"}""",
            request.body.readUtf8()
        )
    }

    @Test
    fun reportsAnOldServerWithoutTheProfileEndpoint() = runTest {
        server.enqueue(MockResponse().setResponseCode(404))
        val result = api().currentSessionProfile()
        assertTrue(result is ApiResult.HttpError)
        assertEquals(404, (result as ApiResult.HttpError).code)
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)
}
