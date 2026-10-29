package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CinecatApi
import io.github.catlinux.cinecat.core.api.PlayResponse
import java.io.IOException

class PlaybackRepository(
    private val apiProvider: () -> CinecatApi?
) {
    suspend fun play(
        id: Long,
        caps: String? = null,
        audio: Int? = null,
        quality: String? = null,
        subtitle: String? = null
    ): ApiResult<PlayResponse> = withApi {
        it.play(id, caps, audio, quality, subtitle)
    }

    suspend fun saveProgress(
        id: Long,
        position: Double,
        duration: Double
    ): ApiResult<Unit> = withApi {
        it.saveProgress(id, position, duration)
    }

    private suspend fun <T> withApi(block: suspend (CinecatApi) -> ApiResult<T>): ApiResult<T> {
        val api = apiProvider()
            ?: return ApiResult.NetworkError(IOException("No hay servidor configurado"))
        return block(api)
    }
}
