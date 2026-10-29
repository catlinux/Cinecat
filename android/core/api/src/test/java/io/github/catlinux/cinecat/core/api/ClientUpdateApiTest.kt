package io.github.catlinux.cinecat.core.api

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class ClientUpdateApiTest {
    @get:Rule
    val server = MockWebServer()

    private fun api(): CinecatApi = createApi(server.url("/"), OkHttpClient())

    @Test
    fun parsesClientUpdateAndBuildsAndroidQuery() = runTest {
        server.enqueue(
            jsonResponse(
                """{
                  "version":"0.34.1",
                  "version_code":3401,
                  "url":"https://github.com/catlinux/Cinecat/releases/download/v0.34.1/cinecat-0.34.1.apk",
                  "sha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
                  "notes":"Novedades",
                  "desconocido":true
                }"""
            )
        )

        val result = api().clientUpdate(3400)

        assertTrue(result is ApiResult.Success)
        val update = (result as ApiResult.Success).value
        requireNotNull(update)
        assertEquals("0.34.1", update.version)
        assertEquals(3401L, update.versionCode)
        assertEquals(
            "https://github.com/catlinux/Cinecat/releases/download/v0.34.1/cinecat-0.34.1.apk",
            update.url
        )
        assertEquals("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", update.sha256)
        assertEquals("Novedades", update.notes)

        val request = server.takeRequest()
        assertEquals("/api/v1/system/client-update", request.requestUrl?.encodedPath)
        assertEquals("android", request.requestUrl?.queryParameter("platform"))
        assertEquals("3400", request.requestUrl?.queryParameter("version_code"))
    }

    @Test
    fun noContentMeansNoUpdate() = runTest {
        server.enqueue(MockResponse().setResponseCode(204))

        val result = api().clientUpdate(3401)

        assertEquals(ApiResult.Success(null), result)
        assertEquals("/api/v1/system/client-update?platform=android&version_code=3401", server.takeRequest().path)
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)
}
