package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.Season
import io.github.catlinux.cinecat.core.api.SeasonDetail
import io.github.catlinux.cinecat.core.api.Series
import io.github.catlinux.cinecat.core.api.SeriesDetail
import io.github.catlinux.cinecat.core.api.SimilarSeriesResponse
import io.github.catlinux.cinecat.core.api.UserList
import io.github.catlinux.cinecat.core.data.LibraryRepository
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

sealed interface SeasonEpisodesState {
    data object Loading : SeasonEpisodesState
    data object Error : SeasonEpisodesState
    data object None : SeasonEpisodesState
    data class Ready(val detail: SeasonDetail) : SeasonEpisodesState
}

sealed interface SeriesDetailUiState {
    data object Loading : SeriesDetailUiState
    data object Error : SeriesDetailUiState
    data class Ready(
        val series: SeriesDetail,
        val seasons: List<Season>,
        val selectedSeason: Int?,
        val episodes: SeasonEpisodesState,
        val similar: List<Series>
    ) : SeriesDetailUiState {
        val seriesWatched: Boolean
            get() = series.episodeCount > 0 && series.watchedCount >= series.episodeCount

        val selectedSeasonInfo: Season?
            get() = seasons.firstOrNull { it.season == selectedSeason }

        val episodeItems: List<MediaItem>
            get() = (episodes as? SeasonEpisodesState.Ready)?.detail?.items.orEmpty()

        val similarSeries: List<Series> get() = similar
        val seasonEpisodes: SeasonEpisodesState get() = episodes
        val selectedSeasonNumber: Int? get() = selectedSeason
    }
}

class SeriesDetailViewModel(
    val id: Long,
    private val loadSeries: suspend () -> ApiResult<SeriesDetail>,
    private val loadSeason: suspend (Int) -> ApiResult<SeasonDetail>,
    private val loadSimilar: suspend () -> ApiResult<SimilarSeriesResponse>,
    private val setSeriesList: suspend (UserList) -> ApiResult<Unit>,
    private val clearSeriesList: suspend (UserList) -> ApiResult<Unit>,
    private val setSeriesWatched: suspend () -> ApiResult<Unit>,
    private val clearSeriesWatched: suspend () -> ApiResult<Unit>,
    private val setSeasonWatched: suspend (Int) -> ApiResult<Unit>,
    private val clearSeasonWatched: suspend (Int) -> ApiResult<Unit>,
    private val setEpisodeWatched: suspend (Long) -> ApiResult<Unit>,
    private val clearEpisodeWatched: suspend (Long) -> ApiResult<Unit>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(id: Long, repository: LibraryRepository, scope: CoroutineScope) : this(
        id = id,
        loadSeries = { repository.seriesDetail(id) },
        loadSeason = { season -> repository.season(id, season) },
        loadSimilar = { repository.seriesSimilar(id, limit = 20) },
        setSeriesList = { list -> repository.setSeriesList(id, list) },
        clearSeriesList = { list -> repository.clearSeriesList(id, list) },
        setSeriesWatched = { repository.setSeriesWatched(id) },
        clearSeriesWatched = { repository.clearSeriesWatched(id) },
        setSeasonWatched = { season -> repository.setSeasonWatched(id, season) },
        clearSeasonWatched = { season -> repository.clearSeasonWatched(id, season) },
        setEpisodeWatched = { itemId -> repository.setWatched(itemId) },
        clearEpisodeWatched = { itemId -> repository.clearWatched(itemId) },
        scope = scope
    )

    private val _state = MutableStateFlow<SeriesDetailUiState>(SeriesDetailUiState.Loading)
    val state: StateFlow<SeriesDetailUiState> = _state.asStateFlow()

    private val eventChannel = Channel<DetailAction>(Channel.BUFFERED)
    val events: Flow<DetailAction> = eventChannel.receiveAsFlow()

    private val seasonCache = mutableMapOf<Int, SeasonDetail>()
    private var loadJob: Job? = null
    private var seasonJob: Job? = null
    private var favoriteJob: Job? = null
    private var watchlistJob: Job? = null
    private var seriesWatchedJob: Job? = null
    private var seasonWatchedJob: Job? = null
    private val episodeJobs = mutableMapOf<Long, Job>()

    init {
        load()
    }

    fun load() {
        loadJob?.cancel()
        seasonJob?.cancel()
        seasonCache.clear()
        _state.value = SeriesDetailUiState.Loading
        loadJob = scope.launch {
            val similarDeferred = async { safe { loadSimilar() } }
            val seriesResult = safe { loadSeries() }
            when (seriesResult) {
                is ApiResult.Success -> {
                    val similar = when (val result = similarDeferred.await()) {
                        is ApiResult.Success -> result.value.series
                        is ApiResult.HttpError, is ApiResult.NetworkError -> emptyList()
                    }
                    val series = seriesResult.value
                    val seasons = sortedSeasons(series.seasons)
                    val selected = chooseInitialSeason(series, seasons)
                    _state.value = SeriesDetailUiState.Ready(
                        series = series,
                        seasons = seasons,
                        selectedSeason = selected,
                        episodes = if (selected == null) {
                            SeasonEpisodesState.None
                        } else {
                            SeasonEpisodesState.Loading
                        },
                        similar = similar
                    )
                    selected?.let { loadSeasonFor(it, force = true) }
                }
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    similarDeferred.cancel()
                    _state.value = SeriesDetailUiState.Error
                }
            }
        }
    }

    fun retry() {
        load()
    }

    fun selectSeason(season: Int) {
        val ready = _state.value as? SeriesDetailUiState.Ready ?: return
        if (ready.selectedSeason == season && ready.episodes is SeasonEpisodesState.Loading) {
            return
        }
        val cached = seasonCache[season]
        if (cached != null) {
            seasonJob?.cancel()
            _state.value = ready.copy(
                selectedSeason = season,
                episodes = SeasonEpisodesState.Ready(cached)
            )
            return
        }
        loadSeasonFor(season, force = false)
    }

    fun retrySeason() {
        val ready = _state.value as? SeriesDetailUiState.Ready ?: return
        ready.selectedSeason?.let { loadSeasonFor(it, force = true) }
    }

    fun toggleFavorite() {
        val ready = _state.value as? SeriesDetailUiState.Ready ?: return
        if (favoriteJob?.isActive == true) {
            return
        }
        val previous = ready.series.favorite
        val next = !previous
        _state.value = ready.copy(series = ready.series.copy(favorite = next))
        favoriteJob = scope.launch {
            val result = safe {
                if (next) {
                    setSeriesList(UserList.Favorite)
                } else {
                    clearSeriesList(UserList.Favorite)
                }
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is SeriesDetailUiState.Ready) {
                            current.copy(series = current.series.copy(favorite = previous))
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Favorite)
                }
            }
        }
    }

    fun toggleWatchlist() {
        val ready = _state.value as? SeriesDetailUiState.Ready ?: return
        if (watchlistJob?.isActive == true) {
            return
        }
        val previous = ready.series.watchlist
        val next = !previous
        _state.value = ready.copy(series = ready.series.copy(watchlist = next))
        watchlistJob = scope.launch {
            val result = safe {
                if (next) {
                    setSeriesList(UserList.Watchlist)
                } else {
                    clearSeriesList(UserList.Watchlist)
                }
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is SeriesDetailUiState.Ready) {
                            current.copy(series = current.series.copy(watchlist = previous))
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Watchlist)
                }
            }
        }
    }

    fun toggleSeriesWatched() {
        val ready = _state.value as? SeriesDetailUiState.Ready ?: return
        if (seriesWatchedJob?.isActive == true || ready.series.episodeCount <= 0) {
            return
        }
        val previous = ready.series.watchedCount
        val next = !ready.seriesWatched
        val optimisticCount = if (next) ready.series.episodeCount else 0L
        _state.value = ready.copy(series = ready.series.copy(watchedCount = optimisticCount))
        seriesWatchedJob = scope.launch {
            val result = safe {
                if (next) setSeriesWatched() else clearSeriesWatched()
            }
            when (result) {
                is ApiResult.Success -> reloadSelectedSeason()
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is SeriesDetailUiState.Ready) {
                            current.copy(
                                series = current.series.copy(watchedCount = previous)
                            )
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Watched)
                }
            }
        }
    }

    fun toggleSeasonWatched() {
        val ready = _state.value as? SeriesDetailUiState.Ready ?: return
        val selected = ready.selectedSeasonInfo ?: return
        if (seasonWatchedJob?.isActive == true || selected.episodeCount <= 0) {
            return
        }
        val next = !isSeasonWatched(selected)
        val previousSeasons = ready.seasons
        val previousEpisodes = ready.episodes
        val targetSeason = selected.season
        _state.value = ready.copy(
            seasons = ready.seasons.map { season ->
                if (season.season == targetSeason) {
                    season.copy(
                        watchedCount = if (next) season.episodeCount else 0L
                    )
                } else {
                    season
                }
            },
            episodes = markEpisodes(ready.episodes, next)
        )
        seasonWatchedJob = scope.launch {
            val result = safe {
                if (next) setSeasonWatched(targetSeason) else clearSeasonWatched(targetSeason)
            }
            when (result) {
                is ApiResult.Success -> reloadSelectedSeason()
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is SeriesDetailUiState.Ready) {
                            if (current.selectedSeason == targetSeason) {
                                current.copy(
                                    seasons = previousSeasons,
                                    episodes = previousEpisodes
                                )
                            } else {
                                val previousSeason = previousSeasons.firstOrNull {
                                    it.season == targetSeason
                                }
                                current.copy(
                                    seasons = current.seasons.map { seasonInfo ->
                                        if (seasonInfo.season == targetSeason &&
                                            previousSeason != null
                                        ) {
                                            previousSeason
                                        } else {
                                            seasonInfo
                                        }
                                    }
                                )
                            }
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Watched)
                }
            }
        }
    }

    fun toggleEpisodeWatched(itemId: Long) {
        val ready = _state.value as? SeriesDetailUiState.Ready ?: return
        val season = ready.episodes as? SeasonEpisodesState.Ready ?: return
        val item = season.detail.items.firstOrNull { it.id == itemId } ?: return
        if (episodeJobs[itemId]?.isActive == true) {
            return
        }
        val selected = ready.selectedSeasonInfo ?: return
        val previousWatched = item.watched
        val next = !previousWatched
        val previousSeasonWatched = selected.watchedCount
        val optimisticCount = (selected.watchedCount + if (next) 1 else -1)
            .coerceIn(0L, selected.episodeCount)
        val updatedDetail = season.detail.copy(
            items = season.detail.items.map { episode ->
                if (episode.id == itemId) episode.copy(watched = next) else episode
            }
        )
        seasonCache[selected.season] = updatedDetail
        _state.value = ready.copy(
            seasons = ready.seasons.map { seasonInfo ->
                if (seasonInfo.season == selected.season) {
                    seasonInfo.copy(watchedCount = optimisticCount)
                } else {
                    seasonInfo
                }
            },
            episodes = SeasonEpisodesState.Ready(updatedDetail)
        )
        episodeJobs[itemId] = scope.launch {
            val result = safe {
                if (next) setEpisodeWatched(itemId) else clearEpisodeWatched(itemId)
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is SeriesDetailUiState.Ready) {
                            val sameSeason = current.selectedSeason == selected.season
                            val currentEpisodes = current.episodes as? SeasonEpisodesState.Ready
                            val currentDetail = currentEpisodes?.detail
                            val rolledBackDetail = if (sameSeason) {
                                currentDetail?.copy(
                                    items = currentDetail.items.map { episode ->
                                        if (episode.id == itemId) {
                                            episode.copy(watched = previousWatched)
                                        } else {
                                            episode
                                        }
                                    }
                                )
                            } else {
                                null
                            }
                            val rolledBackSeasons = current.seasons.map { seasonInfo ->
                                if (seasonInfo.season == selected.season) {
                                    seasonInfo.copy(watchedCount = previousSeasonWatched)
                                } else {
                                    seasonInfo
                                }
                            }
                            if (rolledBackDetail != null) {
                                seasonCache[selected.season] = rolledBackDetail
                            }
                            current.copy(
                                seasons = rolledBackSeasons,
                                episodes = rolledBackDetail?.let(SeasonEpisodesState::Ready)
                                    ?: current.episodes
                            )
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Watched)
                }
            }
        }
    }

    private fun loadSeasonFor(season: Int, force: Boolean) {
        if (force) {
            seasonCache.remove(season)
        }
        if (!force) {
            seasonCache[season]?.let { cached ->
                val current = _state.value as? SeriesDetailUiState.Ready ?: return
                seasonJob?.cancel()
                _state.value = current.copy(
                    selectedSeason = season,
                    episodes = SeasonEpisodesState.Ready(cached)
                )
                return
            }
        }
        val current = _state.value as? SeriesDetailUiState.Ready ?: return
        seasonJob?.cancel()
        _state.value = current.copy(
            selectedSeason = season,
            episodes = SeasonEpisodesState.Loading
        )
        seasonJob = scope.launch {
            val result = safe { loadSeason(season) }
            when (result) {
                is ApiResult.Success -> {
                    seasonCache[season] = result.value
                    _state.update { latest ->
                        if (latest is SeriesDetailUiState.Ready &&
                            latest.selectedSeason == season
                        ) {
                            latest.copy(episodes = SeasonEpisodesState.Ready(result.value))
                        } else {
                            latest
                        }
                    }
                }
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { latest ->
                        if (latest is SeriesDetailUiState.Ready &&
                            latest.selectedSeason == season
                        ) {
                            latest.copy(episodes = SeasonEpisodesState.Error)
                        } else {
                            latest
                        }
                    }
                }
            }
        }
    }

    private fun reloadSelectedSeason() {
        val selected = (_state.value as? SeriesDetailUiState.Ready)?.selectedSeason ?: return
        loadSeasonFor(selected, force = true)
    }

    private fun markEpisodes(
        state: SeasonEpisodesState,
        watched: Boolean
    ): SeasonEpisodesState {
        val ready = state as? SeasonEpisodesState.Ready ?: return state
        return SeasonEpisodesState.Ready(
            ready.detail.copy(items = ready.detail.items.map { it.copy(watched = watched) })
        )
    }

    private fun chooseInitialSeason(series: SeriesDetail, seasons: List<Season>): Int? {
        val nextSeason = series.nextEpisode?.season
        if (nextSeason != null && seasons.any { it.season == nextSeason }) {
            return nextSeason
        }
        return seasons.filter { it.season > 0 }.minByOrNull { it.season }?.season
            ?: seasons.minByOrNull { it.season }?.season
    }

    private fun sortedSeasons(seasons: List<Season>): List<Season> =
        seasons.sortedWith(compareBy({ it.season == 0 }, { it.season }))

    private fun isSeasonWatched(season: Season): Boolean =
        season.episodeCount > 0 && season.watchedCount >= season.episodeCount

    private suspend fun <T> safe(block: suspend () -> ApiResult<T>): ApiResult<T> {
        return try {
            block()
        } catch (error: CancellationException) {
            throw error
        } catch (error: Exception) {
            ApiResult.NetworkError(IOException(error.message, error))
        }
    }

    override fun onCleared() {
        loadJob?.cancel()
        seasonJob?.cancel()
        favoriteJob?.cancel()
        watchlistJob?.cancel()
        seriesWatchedJob?.cancel()
        seasonWatchedJob?.cancel()
        episodeJobs.values.forEach { it.cancel() }
    }
}
