package io.github.catlinux.cinecat.core.api

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class PlaybackApiTest {
    @get:Rule
    val server = MockWebServer()

    private fun api(): CinecatApi = createApi(server.url("/"), OkHttpClient())

    @Test
    fun parsesDirectAndRemuxResponsesAndIgnoresUnknownFields() = runTest {
        server.enqueue(jsonResponse(directJson()))
        val direct = api().play(7)
        assertTrue(direct is ApiResult.Success)
        val directValue = (direct as ApiResult.Success).value
        assertEquals("direct", directValue.mode)
        assertNull(directValue.reason)
        assertEquals("/api/v1/stream/7?exp=1&sig=abc", directValue.url)
        assertEquals(0, directValue.selectedAudio)
        assertEquals("auto", directValue.selectedQuality)
        assertNull(directValue.selectedSubtitle)
        assertEquals("Español", directValue.audioTracks.single().languageName)
        assertTrue(directValue.audioTracks.single().default)
        assertEquals("Auto", directValue.qualities.single().label)
        assertEquals("e0", directValue.subtitles.single().id)
        assertEquals("/api/v1/stream/7/subtitles/e0.vtt", directValue.subtitles.single().url)
        assertTrue(directValue.segments.isEmpty())
        assertEquals("/api/v1/items/7/play", server.takeRequest().path)

        server.enqueue(jsonResponse(remuxJson()))
        val remux = api().play(7)
        assertTrue(remux is ApiResult.Success)
        val remuxValue = (remux as ApiResult.Success).value
        assertEquals("remux", remuxValue.mode)
        assertEquals("audio", remuxValue.reason)
        assertEquals("e0", remuxValue.selectedSubtitle)
        assertEquals("/api/v1/items/7/play", server.takeRequest().path)
    }

    @Test
    fun parsesAutoplayAndNextEpisodeWithDefaults() = runTest {
        server.enqueue(
            jsonResponse(
                """{
                  "url":"/stream",
                  "mode":"direct",
                  "autoplay_next":false,
                  "next_episode":{"id":9,"title":"El siguiente","season":2,"episode":3}
                }"""
            )
        )
        val explicit = api().play(7) as ApiResult.Success
        assertFalse(explicit.value.autoplayNext)
        assertEquals(9L, explicit.value.nextEpisode?.id)
        assertEquals("El siguiente", explicit.value.nextEpisode?.title)
        assertEquals(2, explicit.value.nextEpisode?.season)
        assertEquals(3, explicit.value.nextEpisode?.episode)

        server.enqueue(jsonResponse(directJson()))
        val omitted = api().play(7) as ApiResult.Success
        assertTrue(omitted.value.autoplayNext)
        assertNull(omitted.value.nextEpisode)
    }

    @Test
    fun parsesBurnInSubtitles() = runTest {
        server.enqueue(
            jsonResponse(
                """{
                  "url":"/stream",
                  "mode":"remux",
                  "subtitles":[{"id":"e1","burn_in":true,"url":""}],
                  "selected_subtitle":"e1"
                }"""
            )
        )
        val result = api().play(7)
        assertTrue(result is ApiResult.Success)
        val subtitle = (result as ApiResult.Success).value.subtitles.single()
        assertEquals("e1", subtitle.id)
        assertTrue(subtitle.burnIn)
    }

    @Test
    fun parsesSegmentsAndAllowsResponsesWithoutThem() = runTest {
        server.enqueue(
            jsonResponse(
                """{
                  "url":"/stream",
                  "mode":"remux",
                  "segments":[
                    {"kind":"intro","start":10.5,"end":42.0,"source":"chapter"},
                    {"kind":"credits","start":3500,"end":3600,"source":"manual"}
                  ]
                }"""
            )
        )
        val withSegments = api().play(7)
        assertTrue(withSegments is ApiResult.Success)
        val segments = (withSegments as ApiResult.Success).value.segments
        assertEquals(2, segments.size)
        assertEquals("intro", segments[0].kind)
        assertEquals(10.5, segments[0].start, 0.0)
        assertEquals(42.0, segments[0].end, 0.0)
        assertEquals("chapter", segments[0].source)
        assertEquals("credits", segments[1].kind)
        assertEquals("manual", segments[1].source)

        server.enqueue(jsonResponse(directJson()))
        val withoutSegments = api().play(7)
        assertTrue(withoutSegments is ApiResult.Success)
        assertTrue((withoutSegments as ApiResult.Success).value.segments.isEmpty())
    }

    @Test
    fun playBuildsQueryWithOnlyNonNullParameters() = runTest {
        server.enqueue(jsonResponse(directJson()))
        api().play(
            id = 7,
            caps = "mkv,mp4,h264",
            audio = 1,
            quality = "720",
            subtitle = "e0"
        )
        val request = server.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/items/7/play", request.requestUrl?.encodedPath)
        assertEquals("mkv,mp4,h264", request.requestUrl?.queryParameter("caps"))
        assertEquals("1", request.requestUrl?.queryParameter("audio"))
        assertEquals("720", request.requestUrl?.queryParameter("quality"))
        assertEquals("e0", request.requestUrl?.queryParameter("subtitle"))

        server.enqueue(jsonResponse(directJson()))
        api().play(7)
        assertEquals("/api/v1/items/7/play", server.takeRequest().path)
    }

    @Test
    fun saveProgressSendsPutWithJsonBodyAndAcceptsNoContent() = runTest {
        server.enqueue(MockResponse().setResponseCode(204))
        assertTrue(api().saveProgress(7, position = 12.5, duration = 100.0) is ApiResult.Success)

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/items/7/progress", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"position_seconds\":12.5"))
        assertTrue(body.contains("\"duration_seconds\":100.0"))
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)

    private fun directJson(): String =
        """{
          "url":"/api/v1/stream/7?exp=1&sig=abc",
          "mode":"direct",
          "expires_at":"2026-10-30T00:00:00Z",
          "audio_tracks":[{"index":0,"language":"es","language_name":"Español","codec":"aac","channels":2,"channel_layout":"stereo","title":"Español","default":true}],
          "selected_audio":0,
          "qualities":[{"value":"auto","label":"Auto"}],
          "selected_quality":"auto",
          "subtitles":[{"id":"e0","source":"embedded","language":"es","language_name":"Español","title":"","codec":"subrip","forced":false,"default":false,"url":"/api/v1/stream/7/subtitles/e0.vtt"}],
          "selected_subtitle":null,
          "desconocido":true
        }""".trimIndent()

    private fun remuxJson(): String =
        """{
          "url":"/api/v1/stream/7/playlist.m3u8?exp=1&sig=abc&audio=1&q=720",
          "mode":"remux",
          "reason":"audio",
          "expires_at":"2026-10-30T00:00:00Z",
          "audio_tracks":[],
          "selected_audio":1,
          "qualities":[],
          "selected_quality":"720",
          "subtitles":[],
          "selected_subtitle":"e0",
          "desconocido":true
        }""".trimIndent()
}
