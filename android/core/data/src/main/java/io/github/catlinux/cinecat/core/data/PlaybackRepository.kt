package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CinecatApi
import io.github.catlinux.cinecat.core.api.ItemPreferences
import io.github.catlinux.cinecat.core.api.PlayResponse
import io.github.catlinux.cinecat.core.api.PreferencesPatch
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

    suspend fun itemPreferences(id: Long): ApiResult<ItemPreferences> = withApi {
        it.itemPreferences(id)
    }

    suspend fun getItemPreferences(id: Long): ApiResult<ItemPreferences> = itemPreferences(id)

    suspend fun getPreferences(id: Long): ApiResult<ItemPreferences> = itemPreferences(id)

    suspend fun saveItemPreferences(id: Long, patch: PreferencesPatch): ApiResult<Unit> = withApi {
        it.saveItemPreferences(id, patch)
    }

    suspend fun savePreferences(id: Long, patch: PreferencesPatch): ApiResult<Unit> =
        saveItemPreferences(id, patch)

    suspend fun savePreferences(
        id: Long,
        audioIndex: Int? = null,
        subtitleId: String? = null,
        quality: String? = null
    ): ApiResult<Unit> = saveItemPreferences(
        id,
        PreferencesPatch(
            audioIndex = audioIndex,
            subtitleId = subtitleId,
            quality = quality
        )
    )

    private suspend fun <T> withApi(block: suspend (CinecatApi) -> ApiResult<T>): ApiResult<T> {
        val api = apiProvider()
            ?: return ApiResult.NetworkError(IOException("No hay servidor configurado"))
        return block(api)
    }
}
