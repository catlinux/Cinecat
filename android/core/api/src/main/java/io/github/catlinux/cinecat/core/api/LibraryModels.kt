package io.github.catlinux.cinecat.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class HomeResponse(
    @SerialName("continue") val continueWatching: List<MediaItem> = emptyList(),
    @SerialName("next_up") val nextUp: List<MediaItem> = emptyList(),
    @SerialName("recent_movies") val recentMovies: List<MediaItem> = emptyList(),
    @SerialName("recently_watched") val recentlyWatched: List<MediaItem> = emptyList(),
    @SerialName("recent_series") val recentSeries: List<Series> = emptyList(),
    val watchlist: List<HomeListEntry> = emptyList(),
    val favorites: List<HomeListEntry> = emptyList(),
    @SerialName("has_clips") val hasClips: Boolean = false,
    @SerialName("has_music") val hasMusic: Boolean = false,
    val counts: HomeCounts = HomeCounts()
) {
    val continueItems: List<MediaItem> get() = continueWatching
}

@Serializable
data class HomeListEntry(
    val kind: String = "",
    val item: MediaItem? = null,
    val series: Series? = null
)

@Serializable
data class HomeCounts(
    val movies: Long = 0,
    val episodes: Long = 0,
    val series: Long = 0
)

@Serializable
data class MediaItem(
    val id: Long,
    @SerialName("library_id") val libraryId: Long = 0,
    @SerialName("library_kind") val libraryKind: String = "",
    val title: String = "",
    @SerialName("file_title") val fileTitle: String = "",
    val year: Int? = null,
    val kind: String = "",
    val season: Int? = null,
    val episode: Int? = null,
    @SerialName("tmdb_id") val tmdbId: Long? = null,
    val overview: String? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("original_title") val originalTitle: String? = null,
    val tagline: String? = null,
    val genres: List<String> = emptyList(),
    @SerialName("runtime_minutes") val runtimeMinutes: Int? = null,
    val rating: Double? = null,
    val directors: List<String> = emptyList(),
    val cast: List<String> = emptyList(),
    val credits: Credits = Credits(),
    @SerialName("backdrop_url") val backdropUrl: String? = null,
    val watched: Boolean = false,
    val favorite: Boolean = false,
    val watchlist: Boolean = false,
    val progress: PlaybackProgress? = null,
    val collections: List<ItemCollection> = emptyList(),
    @SerialName("series_id") val seriesId: Long? = null,
    @SerialName("series_title") val seriesTitle: String? = null,
    @SerialName("episode_title") val episodeTitle: String? = null,
    @SerialName("still_url") val stillUrl: String? = null,
    @SerialName("series_poster_url") val seriesPosterUrl: String? = null,
    @SerialName("series_backdrop_url") val seriesBackdropUrl: String? = null
)

@Serializable
data class ItemsResponse(
    val items: List<MediaItem> = emptyList(),
    val total: Long = 0,
    val limit: Int = 0,
    val offset: Int = 0
)

@Serializable
data class SimilarItemsResponse(
    val items: List<MediaItem> = emptyList()
)

@Serializable
data class Series(
    val id: Long,
    val title: String = "",
    @SerialName("file_title") val fileTitle: String = "",
    val year: Int? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    val genres: List<String> = emptyList(),
    @SerialName("season_count") val seasonCount: Long = 0,
    @SerialName("episode_count") val episodeCount: Long = 0,
    @SerialName("watched_count") val watchedCount: Long = 0,
    val favorite: Boolean = false,
    val watchlist: Boolean = false
)

@Serializable
data class SeriesResponse(
    val series: List<Series> = emptyList(),
    val total: Long = 0,
    val limit: Int = 0,
    val offset: Int = 0
)

@Serializable
data class SimilarSeriesResponse(
    val series: List<Series> = emptyList()
)

@Serializable
data class SeriesDetail(
    val id: Long,
    val title: String = "",
    @SerialName("file_title") val fileTitle: String = "",
    @SerialName("original_title") val originalTitle: String? = null,
    val year: Int? = null,
    val tagline: String? = null,
    val overview: String? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("backdrop_url") val backdropUrl: String? = null,
    val genres: List<String> = emptyList(),
    val rating: Double? = null,
    val directors: List<String> = emptyList(),
    val cast: List<String> = emptyList(),
    val credits: Credits = Credits(),
    @SerialName("tmdb_id") val tmdbId: Long? = null,
    @SerialName("season_count") val seasonCount: Long = 0,
    @SerialName("episode_count") val episodeCount: Long = 0,
    @SerialName("watched_count") val watchedCount: Long = 0,
    val favorite: Boolean = false,
    val watchlist: Boolean = false,
    val seasons: List<Season> = emptyList(),
    @SerialName("next_episode") val nextEpisode: MediaItem? = null
)

@Serializable
data class Season(
    val season: Int = 0,
    val name: String = "",
    val overview: String? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("episode_count") val episodeCount: Long = 0,
    @SerialName("watched_count") val watchedCount: Long = 0
)

@Serializable
data class SeasonDetail(
    val season: Int = 0,
    val name: String = "",
    val overview: String? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    val items: List<MediaItem> = emptyList()
)

@Serializable
data class Collection(
    val id: Long,
    val name: String = "",
    val overview: String? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("backdrop_url") val backdropUrl: String? = null,
    @SerialName("item_count") val itemCount: Long = 0,
    val source: String = "",
    val edited: Boolean = false,
    val favorite: Boolean = false,
    val watchlist: Boolean = false
)

@Serializable
data class CollectionDetail(
    val id: Long,
    val name: String = "",
    val overview: String? = null,
    @SerialName("poster_url") val posterUrl: String? = null,
    @SerialName("backdrop_url") val backdropUrl: String? = null,
    @SerialName("item_count") val itemCount: Long = 0,
    val source: String = "",
    val edited: Boolean = false,
    val favorite: Boolean = false,
    val watchlist: Boolean = false,
    val items: List<MediaItem> = emptyList()
)

@Serializable
data class CollectionsResponse(
    val collections: List<Collection> = emptyList(),
    val total: Long = 0,
    val limit: Int = 0,
    val offset: Int = 0
)

@Serializable
data class ItemCollection(
    val id: Long,
    val name: String = ""
)

@Serializable
data class Credits(
    val cast: List<CastCredit> = emptyList(),
    val crew: List<CrewCredit> = emptyList()
)

@Serializable
data class CastCredit(
    val name: String = "",
    val character: String = "",
    @SerialName("photo_url") val photoUrl: String = ""
)

@Serializable
data class CrewCredit(
    val name: String = "",
    val job: String = "",
    @SerialName("photo_url") val photoUrl: String = ""
)

@Serializable
data class GenreCount(
    val name: String = "",
    val count: Int = 0
)

@Serializable
data class GenresResponse(
    val genres: List<GenreCount> = emptyList()
)

@Serializable
data class ListTotals(
    val items: Long = 0,
    val series: Long = 0,
    val collections: Long = 0
)

@Serializable
data class ListsResponse(
    val items: List<MediaItem> = emptyList(),
    val series: List<Series> = emptyList(),
    val collections: List<Collection> = emptyList(),
    val totals: ListTotals = ListTotals(),
    val total: Long? = null,
    val limit: Int = 0,
    val offset: Int = 0
)

@Serializable
data class PlaybackProgress(
    @SerialName("position_seconds") val positionSeconds: Double = 0.0,
    @SerialName("duration_seconds") val durationSeconds: Double = 0.0,
    @SerialName("updated_at") val updatedAt: String = ""
)

@Serializable
data class PortalRow(
    val id: String,
    val visible: Boolean = false,
    val title: String = "",
    val kicker: String = ""
)

@Serializable
data class PublicSettings(
    @SerialName("ui.portal_rows") val portalRows: String? = null,
    @SerialName("ui.show_series") val showSeries: Boolean = true
) {
    val portalRowsRaw: String? get() = portalRows
}

enum class UserList(val value: String) {
    Favorite("favorite"),
    Watchlist("watchlist");

    override fun toString(): String = value
}
