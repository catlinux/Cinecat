package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CinecatApi
import io.github.catlinux.cinecat.core.api.CollectionDetail
import io.github.catlinux.cinecat.core.api.CollectionsResponse
import io.github.catlinux.cinecat.core.api.GenresResponse
import io.github.catlinux.cinecat.core.api.HomeResponse
import io.github.catlinux.cinecat.core.api.ItemsResponse
import io.github.catlinux.cinecat.core.api.ListsResponse
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.PortalRow
import io.github.catlinux.cinecat.core.api.PublicSettings
import io.github.catlinux.cinecat.core.api.SeasonDetail
import io.github.catlinux.cinecat.core.api.Series as ApiSeries
import io.github.catlinux.cinecat.core.api.SeriesDetail
import io.github.catlinux.cinecat.core.api.SeriesResponse
import io.github.catlinux.cinecat.core.api.SimilarItemsResponse
import io.github.catlinux.cinecat.core.api.SimilarSeriesResponse
import io.github.catlinux.cinecat.core.api.UserList
import io.github.catlinux.cinecat.core.api.portalRows
import java.io.IOException

sealed interface PortalEntry {
    data class Item(val item: MediaItem) : PortalEntry {
        val value: MediaItem get() = item
    }

    data class Series(val series: ApiSeries) : PortalEntry {
        val value: ApiSeries get() = series
    }
}

data class PortalRowData(
    val id: String,
    val title: String,
    val kicker: String,
    val entries: List<PortalEntry>
) {
    val mediaItems: List<MediaItem>
        get() = entries.filterIsInstance<PortalEntry.Item>().map { it.item }

    val seriesItems: List<ApiSeries>
        get() = entries.filterIsInstance<PortalEntry.Series>().map { it.series }

    val items: List<MediaItem> get() = mediaItems

    val series: List<ApiSeries> get() = seriesItems
}

data class PortalData(val rows: List<PortalRowData>)

class LibraryRepository(
    private val apiProvider: () -> CinecatApi?
) {
    suspend fun home(): ApiResult<HomeResponse> = withApi { it.home() }

    suspend fun publicSettings(): ApiResult<PublicSettings> = withApi { it.publicSettings() }

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
    ): ApiResult<ItemsResponse> = withApi {
        it.items(
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

    suspend fun item(id: Long): ApiResult<MediaItem> = withApi { it.item(id) }

    suspend fun itemSimilar(
        id: Long,
        limit: Int? = null
    ): ApiResult<SimilarItemsResponse> = withApi { it.itemSimilar(id, limit) }

    suspend fun series(
        q: String? = null,
        letter: String? = null,
        genre: String? = null,
        sort: String? = null,
        limit: Int? = null,
        offset: Int? = null
    ): ApiResult<SeriesResponse> = withApi {
        it.series(q = q, letter = letter, genre = genre, sort = sort, limit = limit, offset = offset)
    }

    suspend fun seriesDetail(id: Long): ApiResult<SeriesDetail> = withApi {
        it.seriesDetail(id)
    }

    suspend fun season(id: Long, n: Int): ApiResult<SeasonDetail> = withApi {
        it.season(id, n)
    }

    suspend fun seriesSimilar(
        id: Long,
        limit: Int? = null
    ): ApiResult<SimilarSeriesResponse> = withApi { it.seriesSimilar(id, limit) }

    suspend fun seriesGenres(): ApiResult<GenresResponse> = withApi { it.seriesGenres() }

    suspend fun genres(kind: String? = "movie"): ApiResult<GenresResponse> = withApi {
        it.genres(kind)
    }

    suspend fun collections(
        q: String? = null,
        sort: String? = null,
        limit: Int? = null,
        offset: Int? = null
    ): ApiResult<CollectionsResponse> = withApi {
        it.collections(q = q, sort = sort, limit = limit, offset = offset)
    }

    suspend fun collection(id: Long): ApiResult<CollectionDetail> = withApi {
        it.collection(id)
    }

    suspend fun lists(
        list: UserList,
        kind: String? = null
    ): ApiResult<ListsResponse> = withApi { it.lists(list, kind) }

    suspend fun setItemList(id: Long, list: UserList): ApiResult<Unit> = withApi {
        it.setItemList(id, list)
    }

    suspend fun clearItemList(id: Long, list: UserList): ApiResult<Unit> = withApi {
        it.clearItemList(id, list)
    }

    suspend fun setSeriesList(id: Long, list: UserList): ApiResult<Unit> = withApi {
        it.setSeriesList(id, list)
    }

    suspend fun clearSeriesList(id: Long, list: UserList): ApiResult<Unit> = withApi {
        it.clearSeriesList(id, list)
    }

    suspend fun setCollectionList(id: Long, list: UserList): ApiResult<Unit> = withApi {
        it.setCollectionList(id, list)
    }

    suspend fun clearCollectionList(id: Long, list: UserList): ApiResult<Unit> = withApi {
        it.clearCollectionList(id, list)
    }

    suspend fun setWatched(id: Long): ApiResult<Unit> = withApi { it.setWatched(id) }

    suspend fun clearWatched(id: Long): ApiResult<Unit> = withApi { it.clearWatched(id) }

    suspend fun setSeriesWatched(id: Long): ApiResult<Unit> = withApi {
        it.setSeriesWatched(id)
    }

    suspend fun clearSeriesWatched(id: Long): ApiResult<Unit> = withApi {
        it.clearSeriesWatched(id)
    }

    suspend fun setSeasonWatched(id: Long, n: Int): ApiResult<Unit> = withApi {
        it.setSeasonWatched(id, n)
    }

    suspend fun clearSeasonWatched(id: Long, n: Int): ApiResult<Unit> = withApi {
        it.clearSeasonWatched(id, n)
    }

    suspend fun portal(): ApiResult<PortalData> {
        val api = apiProvider()
            ?: return ApiResult.NetworkError(IOException("No hay servidor configurado"))
        val home = when (val result = api.home()) {
            is ApiResult.Success -> result.value
            is ApiResult.HttpError -> return result
            is ApiResult.NetworkError -> return result
        }
        val settings = when (val result = api.publicSettings()) {
            is ApiResult.Success -> result.value
            is ApiResult.HttpError -> PublicSettings()
            is ApiResult.NetworkError -> PublicSettings()
        }
        return ApiResult.Success(buildPortal(home, settings))
    }

    private suspend fun <T> withApi(block: suspend (CinecatApi) -> ApiResult<T>): ApiResult<T> {
        val api = apiProvider()
            ?: return ApiResult.NetworkError(IOException("No hay servidor configurado"))
        return block(api)
    }

    private fun buildPortal(home: HomeResponse, settings: PublicSettings): PortalData {
        val showSeries = settings.showSeries
        val rows = portalRows(settings).mapNotNull { row ->
            if (!row.visible) {
                return@mapNotNull null
            }
            val entries = entriesFor(row, home, showSeries)
            if (entries.isEmpty()) {
                null
            } else {
                PortalRowData(row.id, row.title, row.kicker, entries)
            }
        }
        return PortalData(rows)
    }

    private fun entriesFor(
        row: PortalRow,
        home: HomeResponse,
        showSeries: Boolean
    ): List<PortalEntry> = when (row.id) {
        "recent_movies" -> home.recentMovies.map { PortalEntry.Item(it) }
        "continue_movies" -> home.continueWatching
            .filter { it.kind != "episode" }
            .map { PortalEntry.Item(it) }
        "watchlist_movies" -> home.watchlist.mapNotNull { entry ->
            if (entry.kind == "item") entry.item?.let { PortalEntry.Item(it) } else null
        }
        "watched_movies" -> home.recentlyWatched.map { PortalEntry.Item(it) }
        "recent_series" -> if (showSeries) {
            home.recentSeries.map { PortalEntry.Series(it) }
        } else {
            emptyList()
        }
        "continue_series" -> home.continueWatching
            .filter { it.kind == "episode" }
            .map { PortalEntry.Item(it) }
        "next_up" -> home.nextUp.map { PortalEntry.Item(it) }
        "watchlist_series" -> if (showSeries) {
            home.watchlist.mapNotNull { entry ->
                if (entry.kind == "series") {
                    entry.series?.let { PortalEntry.Series(it) }
                } else {
                    null
                }
            }
        } else {
            emptyList()
        }
        "favorites" -> home.favorites.mapNotNull { entry ->
            when (entry.kind) {
                "item" -> entry.item?.let { PortalEntry.Item(it) }
                "series" -> if (showSeries) {
                    entry.series?.let { PortalEntry.Series(it) }
                } else {
                    null
                }
                else -> null
            }
        }
        else -> emptyList()
    }
}
