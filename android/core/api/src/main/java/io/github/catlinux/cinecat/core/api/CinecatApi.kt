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
