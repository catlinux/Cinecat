package io.github.catlinux.cinecat.core.api

import java.net.InetAddress
import okhttp3.Dns
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

class LocalAddressDnsTest {
    @get:Rule
    val server = MockWebServer()

    private val localhost: InetAddress = InetAddress.getByAddress(byteArrayOf(127, 0, 0, 1))

    @Test
    fun reachableLocalAddressIsUsedWithoutDnsLookup() {
        server.enqueue(MockResponse().setResponseCode(200).setBody("ok"))
        val dns = LocalAddressDns(
            serverHost = { "cinecat.test" },
            localAddress = { localhost },
            isLocalNetwork = { true },
            port = { server.port },
            fallback = Dns.SYSTEM
        )
        val client = OkHttpClient.Builder().dns(dns).build()
        val request = Request.Builder()
            .url("http://cinecat.test:${server.port}/")
            .build()

        client.newCall(request).execute().use { response ->
            assertEquals(200, response.code)
        }
        assertEquals("cinecat.test:${server.port}", server.takeRequest().getHeader("Host"))
    }

    @Test
    fun failedProbeUsesFallbackAndIsCached() {
        val fallback = RecordingDns(listOf(localhost))
        var probes = 0
        val dns = LocalAddressDns(
            serverHost = { "cinecat.test" },
            localAddress = { localhost },
            isLocalNetwork = { true },
            probe = { _, _ -> probes++; false },
            port = { 443 },
            fallback = fallback
        )

        assertEquals(listOf(localhost), dns.lookup("cinecat.test"))
        assertEquals(listOf(localhost), dns.lookup("cinecat.test"))
        assertEquals(1, probes)
        assertEquals(2, fallback.calls)
    }

    @Test
    fun outsideLocalNetworkDoesNotProbe() {
        val fallback = RecordingDns(listOf(localhost))
        var probes = 0
        val dns = LocalAddressDns(
            serverHost = { "cinecat.test" },
            localAddress = { localhost },
            isLocalNetwork = { false },
            probe = { _, _ -> probes++; true },
            port = { 443 },
            fallback = fallback
        )

        assertEquals(listOf(localhost), dns.lookup("cinecat.test"))
        assertEquals(0, probes)
        assertEquals(1, fallback.calls)
    }

    @Test
    fun differentHostUsesFallback() {
        val fallback = RecordingDns(listOf(localhost))
        var probes = 0
        val dns = LocalAddressDns(
            serverHost = { "cinecat.test" },
            localAddress = { localhost },
            isLocalNetwork = { true },
            probe = { _, _ -> probes++; true },
            port = { 443 },
            fallback = fallback
        )

        assertEquals(listOf(localhost), dns.lookup("other.test"))
        assertEquals(0, probes)
        assertEquals(1, fallback.calls)
    }

    @Test
    fun successfulProbeIsCachedForSixtySecondsAndInvalidated() {
        val fallback = RecordingDns(listOf(localhost))
        var now = 0L
        var probes = 0
        val dns = LocalAddressDns(
            serverHost = { "cinecat.test" },
            localAddress = { localhost },
            isLocalNetwork = { true },
            probe = { _, _ -> probes++; true },
            port = { 443 },
            fallback = fallback,
            clock = { now }
        )

        assertEquals(listOf(localhost), dns.lookup("cinEcat.test"))
        assertEquals(listOf(localhost), dns.lookup("cinecat.test"))
        assertEquals(1, probes)

        now += 60_001
        assertEquals(listOf(localhost), dns.lookup("cinecat.test"))
        assertEquals(2, probes)

        dns.invalidate()
        assertEquals(listOf(localhost), dns.lookup("cinecat.test"))
        assertEquals(3, probes)
        assertEquals(0, fallback.calls)
    }

    private class RecordingDns(private val result: List<InetAddress>) : Dns {
        var calls = 0

        override fun lookup(hostname: String): List<InetAddress> {
            calls++
            return result
        }
    }
}
