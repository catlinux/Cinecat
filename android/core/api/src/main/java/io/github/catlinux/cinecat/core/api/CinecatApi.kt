package io.github.catlinux.cinecat.core.api

import java.io.IOException
import java.util.concurrent.TimeUnit
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import okhttp3.Dns
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.HttpException
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory

val CinecatJson: Json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
    // Go serializa los slices vacíos como null: se toman como el valor por defecto.
    coerceInputValues = true
}

sealed interface ApiResult<out T> {
    data class Success<T>(val value: T) : ApiResult<T>
    data class HttpError(
        val code: Int,
        val message: String?,
        val retryAfterSeconds: Long? = null
    ) : ApiResult<Nothing>

    data class NetworkError(val cause: IOException) : ApiResult<Nothing>
}

class CinecatApi(
    private val service: CinecatService,
    private val json: Json = CinecatJson
) {
    suspend fun healthz(): ApiResult<String> = call {
        service.healthz().use { it.string() }
    }

    suspend fun version(): ApiResult<ServerVersion> = call { service.version() }

    suspend fun login(request: LoginRequest): ApiResult<LoginResponse> = call {
        service.login(request)
    }

    suspend fun logout(): ApiResult<Unit> = call { service.logout() }

    suspend fun me(): ApiResult<CurrentUserResponse> = call { service.me() }

    suspend fun pairCreate(request: PairCreateRequest): ApiResult<PairCreateResponse> = call {
        service.pairCreate(request)
    }

    suspend fun pairToken(request: PairTokenRequest): ApiResult<PairTokenResponse> = call {
        service.pairToken(request)
    }

    suspend fun home(): ApiResult<HomeResponse> = call { service.home() }

    suspend fun publicSettings(): ApiResult<PublicSettings> = call { service.publicSettings() }

    suspend fun items(
        kind: String? = null,
        q: String? = null,
        letter: String? = null,
        genre: String? = null,
        sort: String? = null,
        watched: Boolean? = null,
        inProgress: Boolean? = null,
        collectionId: Long? = null,
        seriesId: Long? = null,
        limit: Int? = null,
        offset: Int? = null
    ): ApiResult<ItemsResponse> = call {
        service.items(
            kind = kind,
            q = q,
            letter = letter,
            genre = genre,
            sort = sort,
            watched = watched,
            inProgress = inProgress,
            collectionId = collectionId,
            seriesId = seriesId,
            limit = limit,
            offset = offset
        )
    }

    suspend fun item(id: Long): ApiResult<MediaItem> = call { service.item(id) }

    suspend fun itemSimilar(
        id: Long,
        limit: Int? = null
    ): ApiResult<SimilarItemsResponse> = call { service.itemSimilar(id, limit) }

    suspend fun series(
        q: String? = null,
        letter: String? = null,
        genre: String? = null,
        sort: String? = null,
        limit: Int? = null,
        offset: Int? = null
    ): ApiResult<SeriesResponse> = call {
        service.series(q = q, letter = letter, genre = genre, sort = sort, limit = limit, offset = offset)
    }

    suspend fun seriesDetail(id: Long): ApiResult<SeriesDetail> = call {
        service.seriesDetail(id)
    }

    suspend fun season(id: Long, n: Int): ApiResult<SeasonDetail> = call {
        service.season(id, n)
    }

    suspend fun seriesSimilar(
        id: Long,
        limit: Int? = null
    ): ApiResult<SimilarSeriesResponse> = call { service.seriesSimilar(id, limit) }

    suspend fun seriesGenres(): ApiResult<GenresResponse> = call { service.seriesGenres() }

    suspend fun genres(kind: String? = "movie"): ApiResult<GenresResponse> = call {
        service.genres(kind)
    }

    suspend fun collections(
        q: String? = null,
        sort: String? = null,
        limit: Int? = null,
        offset: Int? = null
    ): ApiResult<CollectionsResponse> = call {
        service.collections(q = q, sort = sort, limit = limit, offset = offset)
    }

    suspend fun collection(id: Long): ApiResult<CollectionDetail> = call {
        service.collection(id)
    }

    suspend fun lists(
        list: UserList,
        kind: String? = null
    ): ApiResult<ListsResponse> = call { service.lists(list, kind) }

    suspend fun setItemList(id: Long, list: UserList): ApiResult<Unit> = call {
        service.setItemList(id, list)
    }

    suspend fun clearItemList(id: Long, list: UserList): ApiResult<Unit> = call {
        service.clearItemList(id, list)
    }

    suspend fun setSeriesList(id: Long, list: UserList): ApiResult<Unit> = call {
        service.setSeriesList(id, list)
    }

    suspend fun clearSeriesList(id: Long, list: UserList): ApiResult<Unit> = call {
        service.clearSeriesList(id, list)
    }

    suspend fun setCollectionList(id: Long, list: UserList): ApiResult<Unit> = call {
        service.setCollectionList(id, list)
    }

    suspend fun clearCollectionList(id: Long, list: UserList): ApiResult<Unit> = call {
        service.clearCollectionList(id, list)
    }

    suspend fun setWatched(id: Long): ApiResult<Unit> = call { service.setWatched(id) }

    suspend fun clearWatched(id: Long): ApiResult<Unit> = call { service.clearWatched(id) }

    suspend fun setSeriesWatched(id: Long): ApiResult<Unit> = call {
        service.setSeriesWatched(id)
    }

    suspend fun clearSeriesWatched(id: Long): ApiResult<Unit> = call {
        service.clearSeriesWatched(id)
    }

    suspend fun setSeasonWatched(id: Long, n: Int): ApiResult<Unit> = call {
        service.setSeasonWatched(id, n)
    }

    suspend fun clearSeasonWatched(id: Long, n: Int): ApiResult<Unit> = call {
        service.clearSeasonWatched(id, n)
    }

    private suspend fun <T> call(block: suspend () -> T): ApiResult<T> {
        return try {
            ApiResult.Success(block())
        } catch (error: HttpException) {
            ApiResult.HttpError(
                code = error.code(),
                message = errorMessage(error),
                retryAfterSeconds = retryAfterSeconds(error)
            )
        } catch (error: IOException) {
            ApiResult.NetworkError(error)
        } catch (error: SerializationException) {
            // Un proxy o un servidor que no es Cinecat puede responder 200 con otra cosa.
            ApiResult.NetworkError(IOException("Respuesta no válida del servidor", error))
        }
    }

    private fun errorMessage(error: HttpException): String? {
        return runCatching {
            error.response()?.errorBody()?.use { body ->
                json.decodeFromString(ErrorResponse.serializer(), body.string()).error
            }
        }.getOrNull()
    }

    private fun retryAfterSeconds(error: HttpException): Long? {
        return error.response()?.headers()?.get("Retry-After")?.toLongOrNull()
    }
}

fun createApi(baseUrl: HttpUrl, okHttpClient: OkHttpClient): CinecatApi {
    val normalizedBaseUrl = if (baseUrl.encodedPath.endsWith("/")) {
        baseUrl
    } else {
        baseUrl.newBuilder().addPathSegment("").build()
    }
    val retrofit = Retrofit.Builder()
        .baseUrl(normalizedBaseUrl)
        .client(okHttpClient)
        .addConverterFactory(CinecatJson.asConverterFactory("application/json".toMediaType()))
        .build()
    return CinecatApi(retrofit.create(CinecatService::class.java))
}

fun createApi(baseUrl: String, okHttpClient: OkHttpClient): CinecatApi =
    createApi(baseUrl.toHttpUrl(), okHttpClient)

fun createOkHttpClient(
    serverOrigin: () -> HttpUrl?,
    token: () -> String?,
    onUnauthorized: () -> Unit,
    version: String,
    dns: Dns = Dns.SYSTEM
): OkHttpClient {
    return OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .addInterceptor(UserAgentInterceptor(version))
        .addInterceptor(AuthInterceptor(serverOrigin, token, onUnauthorized))
        .dns(dns)
        .build()
}
