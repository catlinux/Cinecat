package io.github.catlinux.cinecat.core.data

import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

object ServerAddress {
    fun parse(input: String): Result<HttpUrl> {
        val trimmed = input.trim()
        if (trimmed.isEmpty()) {
            return Result.failure(IllegalArgumentException("La dirección del servidor está vacía"))
        }
        val candidate = if ("://" in trimmed) trimmed else "https://$trimmed"
        val parsed = candidate.toHttpUrlOrNull()
            ?: return Result.failure(IllegalArgumentException("La dirección del servidor no es válida"))
        if (!parsed.scheme.equals("https", ignoreCase = true)) {
            return Result.failure<HttpUrl>(IllegalArgumentException("El servidor debe usar HTTPS"))
        }
        if (parsed.encodedUsername.isNotEmpty() || parsed.encodedPassword.isNotEmpty()) {
            return Result.failure(IllegalArgumentException("La dirección no puede incluir credenciales"))
        }
        if (parsed.encodedQuery != null || parsed.encodedFragment != null) {
            return Result.failure(IllegalArgumentException("La dirección no puede incluir consulta ni fragmento"))
        }
        if (parsed.encodedPath != "/" && parsed.encodedPath.isNotEmpty()) {
            return Result.failure(IllegalArgumentException("La dirección debe ser solo el origen del servidor"))
        }
        return Result.success(
            parsed.newBuilder()
                .encodedPath("/")
                .query(null)
                .fragment(null)
                .build()
        )
    }
}
