package io.github.catlinux.cinecat.core.api

import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.Socket
import okhttp3.Dns

class LocalAddressDns(
    private val serverHost: () -> String?,
    private val localAddress: () -> InetAddress?,
    private val isLocalNetwork: () -> Boolean,
    private val probe: (InetAddress, Int) -> Boolean = { address, port -> tcpProbe(address, port) },
    private val port: () -> Int,
    private val fallback: Dns = Dns.SYSTEM,
    private val clock: () -> Long = System::currentTimeMillis
) : Dns {
    private data class CacheKey(val address: InetAddress, val port: Int)
    private data class CacheEntry(val checkedAt: Long, val success: Boolean)

    private val lock = Any()
    private val cache = mutableMapOf<CacheKey, CacheEntry>()

    override fun lookup(hostname: String): List<InetAddress> {
        val configuredHost = serverHost()
        if (configuredHost == null || !hostname.equals(configuredHost, ignoreCase = true)) {
            return fallback.lookup(hostname)
        }
        if (!isLocalNetwork()) {
            return fallback.lookup(hostname)
        }
        val address = localAddress() ?: return fallback.lookup(hostname)
        val targetPort = port()
        val key = CacheKey(address, targetPort)
        val now = clock()
        val cached = synchronized(lock) { cache[key] }
        if (cached != null && now - cached.checkedAt < CACHE_TTL_MS) {
            return if (cached.success) listOf(address) else fallback.lookup(hostname)
        }

        val success = probe(address, targetPort)
        synchronized(lock) {
            cache[key] = CacheEntry(now, success)
        }
        return if (success) listOf(address) else fallback.lookup(hostname)
    }

    fun invalidate() {
        synchronized(lock) {
            cache.clear()
        }
    }

    private companion object {
        const val CACHE_TTL_MS = 60_000L
    }
}

fun tcpProbe(address: InetAddress, port: Int, timeoutMs: Int = 1500): Boolean {
    return try {
        Socket().use { socket ->
            socket.connect(InetSocketAddress(address, port), timeoutMs)
            true
        }
    } catch (_: Exception) {
        false
    }
}
