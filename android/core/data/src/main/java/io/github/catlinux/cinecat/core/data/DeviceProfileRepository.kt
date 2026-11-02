package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CinecatApi
import io.github.catlinux.cinecat.core.api.SessionProfile
import java.io.IOException

class DeviceProfileRepository(
    private val apiProvider: () -> CinecatApi?
) {
    suspend fun currentProfile(): ApiResult<SessionProfile> = withApi {
        it.currentSessionProfile()
    }

    suspend fun saveCurrentProfile(profile: SessionProfile): ApiResult<SessionProfile> =
        withApi { it.saveCurrentSessionProfile(profile) }

    private suspend fun <T> withApi(block: suspend (CinecatApi) -> ApiResult<T>): ApiResult<T> {
        val api = apiProvider()
            ?: return ApiResult.NetworkError(IOException("No hay servidor configurado"))
        return block(api)
    }
}
