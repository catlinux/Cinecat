package io.github.catlinux.cinecat.core.api

import okhttp3.ResponseBody
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

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

    @GET("api/v1/home")
    suspend fun home(): HomeResponse

    @GET("api/v1/settings/public")
    suspend fun publicSettings(): PublicSettings

    @GET("api/v1/items")
    suspend fun items(
        @Query("kind") kind: String?,
        @Query("q") q: String?,
        @Query("letter") letter: String?,
        @Query("genre") genre: String?,
        @Query("sort") sort: String?,
        @Query("watched") watched: Boolean?,
        @Query("in_progress") inProgress: Boolean?,
        @Query("collection_id") collectionId: Long?,
        @Query("series_id") seriesId: Long?,
        @Query("limit") limit: Int?,
        @Query("offset") offset: Int?
    ): ItemsResponse

    @GET("api/v1/items/{id}")
    suspend fun item(@Path("id") id: Long): MediaItem

    @GET("api/v1/items/{id}/similar")
    suspend fun itemSimilar(
        @Path("id") id: Long,
        @Query("limit") limit: Int?
    ): SimilarItemsResponse

    @GET("api/v1/series")
    suspend fun series(
        @Query("q") q: String?,
        @Query("letter") letter: String?,
        @Query("genre") genre: String?,
        @Query("sort") sort: String?,
        @Query("limit") limit: Int?,
        @Query("offset") offset: Int?
    ): SeriesResponse

    @GET("api/v1/series/{id}")
    suspend fun seriesDetail(@Path("id") id: Long): SeriesDetail

    @GET("api/v1/series/{id}/seasons/{n}")
    suspend fun season(
        @Path("id") id: Long,
        @Path("n") season: Int
    ): SeasonDetail

    @GET("api/v1/series/{id}/similar")
    suspend fun seriesSimilar(
        @Path("id") id: Long,
        @Query("limit") limit: Int?
    ): SimilarSeriesResponse

    @GET("api/v1/series/genres")
    suspend fun seriesGenres(): GenresResponse

    @GET("api/v1/genres")
    suspend fun genres(@Query("kind") kind: String?): GenresResponse

    @GET("api/v1/collections")
    suspend fun collections(
        @Query("q") q: String?,
        @Query("sort") sort: String?,
        @Query("limit") limit: Int?,
        @Query("offset") offset: Int?
    ): CollectionsResponse

    @GET("api/v1/collections/{id}")
    suspend fun collection(@Path("id") id: Long): CollectionDetail

    @GET("api/v1/lists/{list}")
    suspend fun lists(
        @Path("list") list: UserList,
        @Query("kind") kind: String?
    ): ListsResponse

    @PUT("api/v1/items/{id}/lists/{list}")
    suspend fun setItemList(
        @Path("id") id: Long,
        @Path("list") list: UserList
    )

    @DELETE("api/v1/items/{id}/lists/{list}")
    suspend fun clearItemList(
        @Path("id") id: Long,
        @Path("list") list: UserList
    )

    @PUT("api/v1/series/{id}/lists/{list}")
    suspend fun setSeriesList(
        @Path("id") id: Long,
        @Path("list") list: UserList
    )

    @DELETE("api/v1/series/{id}/lists/{list}")
    suspend fun clearSeriesList(
        @Path("id") id: Long,
        @Path("list") list: UserList
    )

    @PUT("api/v1/collections/{id}/lists/{list}")
    suspend fun setCollectionList(
        @Path("id") id: Long,
        @Path("list") list: UserList
    )

    @DELETE("api/v1/collections/{id}/lists/{list}")
    suspend fun clearCollectionList(
        @Path("id") id: Long,
        @Path("list") list: UserList
    )

    @GET("api/v1/items/{id}/play")
    suspend fun play(
        @Path("id") id: Long,
        @Query("caps") caps: String?,
        @Query("audio") audio: Int?,
        @Query("quality") quality: String?,
        @Query("subtitle") subtitle: String?
    ): PlayResponse

    @PUT("api/v1/items/{id}/progress")
    suspend fun saveProgress(
        @Path("id") id: Long,
        @Body request: ProgressRequest
    )

    @PUT("api/v1/items/{id}/watched")
    suspend fun setWatched(@Path("id") id: Long)

    @DELETE("api/v1/items/{id}/watched")
    suspend fun clearWatched(@Path("id") id: Long)

    @PUT("api/v1/series/{id}/watched")
    suspend fun setSeriesWatched(@Path("id") id: Long)

    @DELETE("api/v1/series/{id}/watched")
    suspend fun clearSeriesWatched(@Path("id") id: Long)

    @PUT("api/v1/series/{id}/seasons/{n}/watched")
    suspend fun setSeasonWatched(
        @Path("id") id: Long,
        @Path("n") season: Int
    )

    @DELETE("api/v1/series/{id}/seasons/{n}/watched")
    suspend fun clearSeasonWatched(
        @Path("id") id: Long,
        @Path("n") season: Int
    )
}
