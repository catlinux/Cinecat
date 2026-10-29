package io.github.catlinux.cinecat.core.api

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class CinecatApiTest {
    @get:Rule
    val server = MockWebServer()

    private fun api(): CinecatApi = createApi(server.url("/"), OkHttpClient())

    @Test
    fun parsesEveryEndpointAndIgnoresUnknownFields() = runTest {
        server.enqueue(MockResponse().setResponseCode(200).setBody("ok"))
        assertEquals(ApiResult.Success("ok"), api().healthz())
        assertEquals("/healthz", server.takeRequest().path)

        server.enqueue(
            jsonResponse(
                """{"server":"0.30.0","commit":"abc1234","built_at":"2026-10-29T00:00:00Z","api_level":1,"min_client":"0.0.0","unknown":"ignored"}"""
            )
        )
        val version = api().version()
        assertTrue(version is ApiResult.Success)
        assertEquals("0.30.0", (version as ApiResult.Success).value.server)
        assertEquals(1, version.value.apiLevel)
        assertEquals("0.0.0", version.value.minClient)
        assertEquals("/api/v1/system/version", server.takeRequest().path)

        server.enqueue(
            jsonResponse(
                """{"token":"token-1","expires_at":"2026-10-30T00:00:00Z","session_id":7,"unknown":true}"""
            )
        )
        val login = api().login(LoginRequest("ana", "secret"))
        assertTrue(login is ApiResult.Success)
        assertEquals("token-1", (login as ApiResult.Success).value.token)
        val loginRequest = server.takeRequest()
        assertEquals("/api/v1/auth/login", loginRequest.path)
        val loginBody = loginRequest.body.readUtf8()
        assertTrue(loginBody.contains("\"username\":\"ana\""))
        assertTrue(loginBody.contains("\"password\":\"secret\""))
        assertTrue(!loginBody.contains("device_name"))

        server.enqueue(MockResponse().setResponseCode(204))
        assertEquals(ApiResult.Success(Unit), api().logout())
        assertEquals("/api/v1/auth/logout", server.takeRequest().path)

        server.enqueue(
            jsonResponse(
                """{"username":"ana","role":"admin","all_libraries":true,"library_ids":[1,2],"unknown":"ignored"}"""
            )
        )
        val me = api().me()
        assertTrue(me is ApiResult.Success)
        assertEquals("ana", (me as ApiResult.Success).value.username)
        assertEquals(listOf(1L, 2L), me.value.libraryIds)
        assertEquals("/api/v1/auth/me", server.takeRequest().path)

        server.enqueue(
            jsonResponse(
                """{"pair_id":"pair-1","secret":"secret-1","code":"ABCD-EFGH","verification_path":"/vincular","expires_in":600,"interval":5,"unknown":"ignored"}"""
            )
        )
        val pair = api().pairCreate(PairCreateRequest("TV", "androidtv"))
        assertTrue(pair is ApiResult.Success)
        assertEquals("ABCD-EFGH", (pair as ApiResult.Success).value.code)
        assertEquals("/api/v1/auth/pair", server.takeRequest().path)

        server.enqueue(
            jsonResponse(
                """{"token":"token-2","expires_at":"2026-10-30T00:00:00Z","session_id":8,"user":{"username":"ana","role":"admin"},"unknown":"ignored"}"""
            )
        )
        val pairToken = api().pairToken(PairTokenRequest("pair-1", "secret-1"))
        assertTrue(pairToken is ApiResult.Success)
        assertEquals("ana", (pairToken as ApiResult.Success).value.user.username)
        assertEquals("/api/v1/auth/pair/token", server.takeRequest().path)
    }

    @Test
    fun mapsHttpErrorsAndRetryAfter() = runTest {
        val cases = listOf(
            401 to "sesión no válida",
            404 to "no encontrado",
            410 to "caducado",
            428 to "pendiente",
            429 to "demasiado rápido"
        )
        for ((code, message) in cases) {
            server.enqueue(
                jsonResponse("""{"error":"$message"}""")
                    .setResponseCode(code)
                    .apply { if (code == 428 || code == 429) addHeader("Retry-After", "5") }
            )
            val result = api().version()
            assertTrue(result is ApiResult.HttpError)
            result as ApiResult.HttpError
            assertEquals(code, result.code)
            assertEquals(message, result.message)
            assertEquals(if (code == 428 || code == 429) 5L else null, result.retryAfterSeconds)
        }
    }

    @Test
    fun mapsNetworkFailure() = runTest {
        val down = MockWebServer()
        down.start()
        val api = createApi(down.url("/"), OkHttpClient())
        down.shutdown()

        val result = api.version()
        assertTrue(result is ApiResult.NetworkError)
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)
}
