package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.createApi
import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class ServerCheckTest {
    @get:Rule
    val server = MockWebServer()

    private val apiFactory: (okhttp3.HttpUrl) -> io.github.catlinux.cinecat.core.api.CinecatApi = { url ->
        createApi(url, OkHttpClient())
    }

    @Test
    fun acceptsCinecatServerWithCompatibleVersion() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("ok"))
        server.enqueue(jsonVersion(apiLevel = 1, minClient = "0.30.0"))

        val result = checkServer(server.url("/"), apiFactory, "0.30.0")

        assertTrue(result is ServerCheckResult.Compatible)
        assertEquals("0.30.0", (result as ServerCheckResult.Compatible).version.server)
    }

    @Test
    fun rejectsServerWithoutHealthzOk() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("not cinecat"))

        assertEquals(ServerCheckResult.NotCinecat, checkServer(server.url("/"), apiFactory, "0.30.0"))
    }

    @Test
    fun reportsIncompatibleVersion() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("ok"))
        server.enqueue(jsonVersion(apiLevel = 2, minClient = "0.30.0"))

        val result = checkServer(server.url("/"), apiFactory, "0.30.0")

        assertTrue(result is ServerCheckResult.Incompatible)
        assertEquals(
            Compatibility.ApiIncompatible,
            (result as ServerCheckResult.Incompatible).reason
        )
    }

    @Test
    fun reportsUnreachableServer() = runTest {
        val down = MockWebServer()
        down.start()
        val address = down.url("/")
        down.shutdown()

        assertEquals(ServerCheckResult.Unreachable, checkServer(address, apiFactory, "0.30.0"))
    }

    private fun jsonVersion(apiLevel: Int, minClient: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(
                """{"server":"0.30.0","commit":"abc","built_at":"2026-10-29T00:00:00Z","api_level":$apiLevel,"min_client":"$minClient"}"""
            )
}
