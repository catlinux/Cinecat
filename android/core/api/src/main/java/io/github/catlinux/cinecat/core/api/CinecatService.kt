package io.github.catlinux.cinecat.core.api

import okhttp3.ResponseBody
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

interface CinecatService {
    @GET("healthz")
    suspend fun healthz(): ResponseBody

    @GET("api/v1/system/version")
    suspend fun version(): ServerVersion

    @POST("api/v1/auth/login")
    suspend fun login(@Body request: LoginRequest): LoginResponse

    @POST("api/v1/auth/logout")
    suspend fun logout()

    @GET("api/v1/auth/me")
    suspend fun me(): CurrentUserResponse

    @POST("api/v1/auth/pair")
    suspend fun pairCreate(@Body request: PairCreateRequest): PairCreateResponse

    @POST("api/v1/auth/pair/token")
    suspend fun pairToken(@Body request: PairTokenRequest): PairTokenResponse
}
