package io.github.catlinux.cinecat.core.data

import java.net.Inet6Address
import java.net.InetAddress

object LocalAddress {
    fun parse(input: String): Result<InetAddress?> {
        val trimmed = input.trim()
        if (trimmed.isEmpty()) {
            return Result.success(null)
        }
        if (trimmed.contains(":")) {
            return parseIpv6(trimmed)
        }
        return parseIpv4(trimmed)
    }

    private fun parseIpv4(value: String): Result<InetAddress> {
        val parts = value.split(".")
        if (parts.size != 4) {
            return invalid()
        }
        val octets = parts.map { part ->
            if (part.isEmpty() || part.length > 3 || part.any { !it.isDigit() }) {
                return invalid()
            }
            part.toIntOrNull()?.takeIf { it in 0..255 } ?: return invalid()
        }
        val isPrivate = octets[0] == 10 ||
            (octets[0] == 172 && octets[1] in 16..31) ||
            (octets[0] == 192 && octets[1] == 168)
        if (!isPrivate) {
            return invalid()
        }
        return Result.success(
            InetAddress.getByAddress(octets.map { it.toByte() }.toByteArray())
        )
    }

    private fun parseIpv6(value: String): Result<InetAddress> {
        if (!value.matches(Regex("^[0-9A-Fa-f:]+$"))) {
            return invalid()
        }
        val address = runCatching { InetAddress.getByName(value) }.getOrNull()
        if (address !is Inet6Address) {
            return invalid()
        }
        val bytes = address.address
        val first = bytes[0].toInt() and 0xff
        val second = bytes[1].toInt() and 0xff
        val uniqueLocal = (first and 0xfe) == 0xfc
        val linkLocal = first == 0xfe && second and 0xc0 == 0x80
        return if (uniqueLocal || linkLocal) {
            Result.success(address)
        } else {
            invalid()
        }
    }

    private fun <T> invalid(): Result<T> =
        Result.failure(IllegalArgumentException("La dirección local no es una IPv4 o IPv6 privada válida"))
}
