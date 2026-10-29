package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CinecatApi
import io.github.catlinux.cinecat.core.api.ServerVersion
import okhttp3.HttpUrl

sealed interface ServerCheckResult {
    data class Compatible(val version: ServerVersion) : ServerCheckResult
    data class Incompatible(val version: ServerVersion, val reason: Compatibility) : ServerCheckResult
    data object NotCinecat : ServerCheckResult
    data object Unreachable : ServerCheckResult
}

suspend fun checkServer(
    address: HttpUrl,
    apiFactory: (HttpUrl) -> CinecatApi,
    appVersion: String
): ServerCheckResult {
    val api = apiFactory(address)
    when (val health = api.healthz()) {
        is ApiResult.Success -> {
            if (!health.value.trim().equals("ok", ignoreCase = true)) {
                return ServerCheckResult.NotCinecat
            }
        }
        is ApiResult.HttpError -> return ServerCheckResult.NotCinecat
        is ApiResult.NetworkError -> return ServerCheckResult.Unreachable
    }

    val version = when (val result = api.version()) {
        is ApiResult.Success -> result.value
        is ApiResult.HttpError -> return ServerCheckResult.NotCinecat
        is ApiResult.NetworkError -> return ServerCheckResult.Unreachable
    }
    val compatibility = checkCompatibility(version, appVersion)
    return if (compatibility == Compatibility.Compatible) {
        ServerCheckResult.Compatible(version)
    } else {
        ServerCheckResult.Incompatible(version, compatibility)
    }
}
