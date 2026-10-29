package io.github.catlinux.cinecat.core.api

import okhttp3.HttpUrl
import okhttp3.Interceptor
import okhttp3.Response

class AuthInterceptor(
    private val serverOrigin: () -> HttpUrl?,
    private val token: () -> String?,
    private val onUnauthorized: () -> Unit
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val original = chain.request()
        val origin = serverOrigin()
        val accessToken = token()?.takeIf { it.isNotBlank() }
        val sameOrigin = origin != null &&
            original.url.scheme.equals(origin.scheme, ignoreCase = true) &&
            original.url.host.equals(origin.host, ignoreCase = true) &&
            original.url.port == origin.port
        val addToken = accessToken != null &&
            sameOrigin &&
            original.header("Authorization") == null
        val request = if (addToken) {
            original.newBuilder()
                .header("Authorization", "Bearer $accessToken")
                .build()
        } else {
            original
        }
        val response = chain.proceed(request)
        if (addToken && response.code == 401) {
            onUnauthorized()
        }
        return response
    }
}

class UserAgentInterceptor(private val version: String) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request().newBuilder()
            .header("User-Agent", "Cinecat-Android/$version")
            .build()
        return chain.proceed(request)
    }
}
