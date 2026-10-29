package io.github.catlinux.cinecat.core.api

import java.net.InetAddress
import okhttp3.Dns
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test

class AuthInterceptorTest {
    @get:Rule
    val server = MockWebServer()

    @Test
    fun addsBearerOnlyToConfiguredOriginAndReportsUnauthorized() {
        var unauthorized = 0
        val client = OkHttpClient.Builder()
            .addInterceptor(
                AuthInterceptor(
                    serverOrigin = { server.url("/") },
                    token = { "token-1" },
                    onUnauthorized = { unauthorized++ }
                )
            )
            .build()

        server.enqueue(MockResponse().setResponseCode(200))
        client.newCall(Request.Builder().url(server.url("/api/v1/auth/me")).build())
            .execute()
            .use { }
        assertEquals("Bearer token-1", server.takeRequest().getHeader("Authorization"))

        val other = MockWebServer()
        other.start()
        try {
            other.enqueue(MockResponse().setResponseCode(200))
            client.newCall(Request.Builder().url(other.url("/api/v1/auth/me")).build())
                .execute()
                .use { }
            assertNull(other.takeRequest().getHeader("Authorization"))
        } finally {
            other.shutdown()
        }

        server.enqueue(MockResponse().setResponseCode(200))
        client.newCall(
            Request.Builder()
                .url(server.url("/api/v1/auth/me"))
                .header("Authorization", "Custom token")
                .build()
        ).execute().use { }
        assertEquals("Custom token", server.takeRequest().getHeader("Authorization"))

        server.enqueue(MockResponse().setResponseCode(401))
        client.newCall(Request.Builder().url(server.url("/api/v1/auth/me")).build())
            .execute()
            .use { }
        assertEquals(1, unauthorized)
    }

    @Test
    fun doesNotAuthenticateToAnotherHostOnTheSamePort() {
        val localhost = InetAddress.getByAddress(byteArrayOf(127, 0, 0, 1))
        val client = OkHttpClient.Builder()
            .dns(object : Dns {
                override fun lookup(hostname: String): List<InetAddress> = listOf(localhost)
            })
            .addInterceptor(
                AuthInterceptor(
                    serverOrigin = { server.url("/") },
                    token = { "token-1" },
                    onUnauthorized = {}
                )
            )
            .build()
        server.enqueue(MockResponse().setResponseCode(200))
        client.newCall(
            Request.Builder()
                .url("http://other.test:${server.port}/api/v1/auth/me")
                .build()
        ).execute().use { }
        assertNull(server.takeRequest().getHeader("Authorization"))
    }

    @Test
    fun doesNotAuthenticateWhenNoTokenIsAvailable() {
        val client = OkHttpClient.Builder()
            .addInterceptor(
                AuthInterceptor(
                    serverOrigin = { server.url("/") },
                    token = { null },
                    onUnauthorized = {}
                )
            )
            .build()
        server.enqueue(MockResponse().setResponseCode(200))
        client.newCall(Request.Builder().url(server.url("/api/v1/auth/me")).build())
            .execute()
            .use { }
        assertNull(server.takeRequest().getHeader("Authorization"))
    }

    @Test
    fun factoryClientCombinesUserAgentAndAuthentication() {
        val client = createOkHttpClient(
            serverOrigin = { server.url("/") },
            token = { "token-1" },
            onUnauthorized = {},
            version = "1.2.3"
        )
        server.enqueue(MockResponse().setResponseCode(200))
        client.newCall(Request.Builder().url(server.url("/api/v1/auth/me")).build())
            .execute()
            .use { }
        val request = server.takeRequest()
        assertEquals("Cinecat-Android/1.2.3", request.getHeader("User-Agent"))
        assertEquals("Bearer token-1", request.getHeader("Authorization"))
    }

    @Test
    fun setsUserAgent() {
        val client = OkHttpClient.Builder()
            .addInterceptor(UserAgentInterceptor("0.30.0"))
            .build()
        server.enqueue(MockResponse().setResponseCode(200))
        client.newCall(Request.Builder().url(server.url("/")).build()).execute().use { }
        assertEquals("Cinecat-Android/0.30.0", server.takeRequest().getHeader("User-Agent"))
    }
}
