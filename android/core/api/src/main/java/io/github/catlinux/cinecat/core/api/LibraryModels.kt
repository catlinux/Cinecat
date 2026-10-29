package io.github.catlinux.cinecat.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class HomeResponse(
    @SerialName("continue") val continueWatching: List<MediaItem>,
    @SerialName("next_up") val nextUp: List<MediaItem>,
    @SerialName("recent_movies") val recentMovies: List<MediaItem>,
    @SerialName("recently_watched") val recentlyWatched: List<MediaItem>,
    @SerialName("recent_series") val recentSeries: List<Series>,
    val watchlist: List<HomeListEntry>,
    val favorites: List<HomeListEntry>,
    @SerialName("has_clips") val hasClips: Boolean,
    @SerialName("has_music") val hasMusic: Boolean,
    val counts: HomeCounts
) {
    val continueItems: List<MediaItem> get() = continueWatching
}

@Serializable
data class HomeListEntry(
    val kind: String,
    val item: MediaItem? = null,
    val series: Series? = null
)

@Serializable
data class HomeCounts(
    val movies: Long,
    val episodes: Long,
    val series: Long
)

@Serializable
data class MediaItem(
    val id: Long,
    @SerialName("library_id") val libraryId: Long,
    @SerialName("library_kind") val libraryKind: String,
    val title: String,
    @SerialName("file_title") val fileTitle: String,
    val year: Int?,
    val kind: String,
    val season: Int?,
    val episode: Int?,
    @SerialName("tmdb_id") val tmdbId: Long?,
    val overview: String?,
    @SerialName("poster_url") val posterUrl: String?,
    @SerialName("original_title") val originalTitle: String?,
    val tagline: String?,
    val genres: List<String>,
    @SerialName("runtime_minutes") val runtimeMinutes: Int?,
    val rating: Double?,
    val directors: List<String>,
    val cast: List<String>,
    val credits: Credits,
    @SerialName("backdrop_url") val backdropUrl: String?,
    val watched: Boolean,
    val favorite: Boolean,
    val watchlist: Boolean,
    val progress: PlaybackProgress?,
    val collections: List<ItemCollection>,
    @SerialName("series_id") val seriesId: Long?,
    @SerialName("series_title") val seriesTitle: String?,
    @SerialName("episode_title") val episodeTitle: String?,
    @SerialName("still_url") val stillUrl: String?,
    @SerialName("series_poster_url") val seriesPosterUrl: String?,
    @SerialName("series_backdrop_url") val seriesBackdropUrl: String?
)

@Serializable
data class ItemsResponse(
    val items: List<MediaItem>,
    val total: Long,
    val limit: Int,
    val offset: Int
)

@Serializable
data class SimilarItemsResponse(
    val items: List<MediaItem>
)

@Serializable
data class Series(
    val id: Long,
    val title: String,
    @SerialName("file_title") val fileTitle: String,
    val year: Int?,
    @SerialName("poster_url") val posterUrl: String?,
    val genres: List<String>,
    @SerialName("season_count") val seasonCount: Long,
    @SerialName("episode_count") val episodeCount: Long,
    @SerialName("watched_count") val watchedCount: Long,
    val favorite: Boolean,
    val watchlist: Boolean
)

@Serializable
data class SeriesResponse(
    val series: List<Series>,
    val total: Long,
    val limit: Int,
    val offset: Int
)

@Serializable
data class SimilarSeriesResponse(
    val series: List<Series>
)

@Serializable
data class SeriesDetail(
    val id: Long,
    val title: String,
    @SerialName("file_title") val fileTitle: String,
    @SerialName("original_title") val originalTitle: String?,
    val year: Int?,
    val tagline: String?,
    val overview: String?,
    @SerialName("poster_url") val posterUrl: String?,
    @SerialName("backdrop_url") val backdropUrl: String?,
    val genres: List<String>,
    val rating: Double?,
    val directors: List<String>,
    val cast: List<String>,
    val credits: Credits,
    @SerialName("tmdb_id") val tmdbId: Long?,
    @SerialName("season_count") val seasonCount: Long,
    @SerialName("episode_count") val episodeCount: Long,
    @SerialName("watched_count") val watchedCount: Long,
    val favorite: Boolean,
    val watchlist: Boolean,
    val seasons: List<Season>,
    @SerialName("next_episode") val nextEpisode: MediaItem?
)

@Serializable
data class Season(
    val season: Int,
    val name: String,
    val overview: String?,
    @SerialName("poster_url") val posterUrl: String?,
    @SerialName("episode_count") val episodeCount: Long,
    @SerialName("watched_count") val watchedCount: Long
)

@Serializable
data class SeasonDetail(
    val season: Int,
    val name: String,
    val overview: String?,
    @SerialName("poster_url") val posterUrl: String?,
    val items: List<MediaItem>
)

@Serializable
data class Collection(
    val id: Long,
    val name: String,
    val overview: String?,
    @SerialName("poster_url") val posterUrl: String?,
    @SerialName("backdrop_url") val backdropUrl: String?,
    @SerialName("item_count") val itemCount: Long,
    val source: String,
    val edited: Boolean,
    val favorite: Boolean,
    val watchlist: Boolean
)

@Serializable
data class CollectionDetail(
    val id: Long,
    val name: String,
    val overview: String?,
    @SerialName("poster_url") val posterUrl: String?,
    @SerialName("backdrop_url") val backdropUrl: String?,
    @SerialName("item_count") val itemCount: Long,
    val source: String,
    val edited: Boolean,
    val favorite: Boolean,
    val watchlist: Boolean,
    val items: List<MediaItem>
)

@Serializable
data class CollectionsResponse(
    val collections: List<Collection>,
    val total: Long,
    val limit: Int,
    val offset: Int
)

@Serializable
data class ItemCollection(
    val id: Long,
    val name: String
)

@Serializable
data class Credits(
    val cast: List<CastCredit>,
    val crew: List<CrewCredit>
)

@Serializable
data class CastCredit(
    val name: String,
    val character: String,
    @SerialName("photo_url") val photoUrl: String
)

@Serializable
data class CrewCredit(
    val name: String,
    val job: String,
    @SerialName("photo_url") val photoUrl: String
)

@Serializable
data class GenreCount(
    val name: String,
    val count: Int
)

@Serializable
data class GenresResponse(
    val genres: List<GenreCount>
)

@Serializable
data class ListTotals(
    val items: Long,
    val series: Long,
    val collections: Long
)

@Serializable
data class ListsResponse(
    val items: List<MediaItem>,
    val series: List<Series>,
    val collections: List<Collection>,
    val totals: ListTotals,
    val total: Long? = null,
    val limit: Int,
    val offset: Int
)

@Serializable
data class PlaybackProgress(
    @SerialName("position_seconds") val positionSeconds: Double,
    @SerialName("duration_seconds") val durationSeconds: Double,
    @SerialName("updated_at") val updatedAt: String
)

@Serializable
data class PortalRow(
    val id: String,
    val visible: Boolean,
    val title: String,
    val kicker: String
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
