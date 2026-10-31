package io.github.catlinux.cinecat.core.api

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class PreferencesApiTest {
    @get:Rule
    val server = MockWebServer()

    private fun api(): CinecatApi = createApi(server.url("/"), OkHttpClient())

    @Test
    fun readsPreferencesAndParsesNulls() = runTest {
        server.enqueue(jsonResponse("""{"audio_index":2,"subtitle_id":"e1","quality":"720"}"""))
        val preferences = api().itemPreferences(7)
        assertTrue(preferences is ApiResult.Success)
        val value = (preferences as ApiResult.Success).value
        assertEquals(2, value.audioIndex)
        assertEquals("e1", value.subtitleId)
        assertEquals("720", value.quality)
        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/items/7/preferences", request.path)

        server.enqueue(
            jsonResponse("""{"audio_index":null,"subtitle_id":null,"quality":null}""")
        )
        val empty = api().itemPreferences(7)
        assertTrue(empty is ApiResult.Success)
        val emptyValue = (empty as ApiResult.Success).value
        assertNull(emptyValue.audioIndex)
        assertNull(emptyValue.subtitleId)
        assertNull(emptyValue.quality)
    }

    @Test
    fun savesOnlyTheProvidedPatchFields() = runTest {
        server.enqueue(MockResponse().setResponseCode(204))
        val audio = api().savePreferences(7, audioIndex = 1, quality = "720")
        assertTrue(audio is ApiResult.Success)
        var request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/items/7/preferences", request.path)
        assertEquals("""{"audio_index":1,"quality":"720"}""", request.body.readUtf8())

        server.enqueue(MockResponse().setResponseCode(204))
        val subtitles = api().savePreferences(7, subtitleId = "")
        assertTrue(subtitles is ApiResult.Success)
        request = server.takeRequest()
        assertEquals("""{"subtitle_id":""}""", request.body.readUtf8())
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)
}
