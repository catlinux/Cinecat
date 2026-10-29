package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.SimilarItemsResponse
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

enum class DetailAction {
    Favorite,
    Watchlist,
    Watched
}

sealed interface MovieDetailUiState {
    data object Loading : MovieDetailUiState
    data object Error : MovieDetailUiState
    data class Ready(
        val item: MediaItem,
        val similar: List<MediaItem>
    ) : MovieDetailUiState {
        val similarItems: List<MediaItem> get() = similar
    }
}

class MovieDetailViewModel(
    val id: Long,
    private val loadItem: suspend () -> ApiResult<MediaItem>,
    private val loadSimilar: suspend () -> ApiResult<SimilarItemsResponse>,
    private val setItemList: suspend (UserList) -> ApiResult<Unit>,
    private val clearItemList: suspend (UserList) -> ApiResult<Unit>,
    private val setWatched: suspend () -> ApiResult<Unit>,
    private val clearWatched: suspend () -> ApiResult<Unit>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(id: Long, repository: LibraryRepository, scope: CoroutineScope) : this(
        id = id,
        loadItem = { repository.item(id) },
        loadSimilar = { repository.itemSimilar(id, limit = 20) },
        setItemList = { list -> repository.setItemList(id, list) },
        clearItemList = { list -> repository.clearItemList(id, list) },
        setWatched = { repository.setWatched(id) },
        clearWatched = { repository.clearWatched(id) },
        scope = scope
    )

    private val _state = MutableStateFlow<MovieDetailUiState>(MovieDetailUiState.Loading)
    val state: StateFlow<MovieDetailUiState> = _state.asStateFlow()

    private val eventChannel = Channel<DetailAction>(Channel.BUFFERED)
    val events: Flow<DetailAction> = eventChannel.receiveAsFlow()

    private var loadJob: Job? = null
    private var favoriteJob: Job? = null
    private var watchlistJob: Job? = null
    private var watchedJob: Job? = null

    init {
        load()
    }

    fun load() {
        loadJob?.cancel()
        _state.value = MovieDetailUiState.Loading
        loadJob = scope.launch {
            val similarDeferred = async { safe { loadSimilar() } }
            val itemResult = safe { loadItem() }
            when (itemResult) {
                is ApiResult.Success -> {
                    val similarResult = similarDeferred.await()
                    val similar = when (similarResult) {
                        is ApiResult.Success -> similarResult.value.items
                        is ApiResult.HttpError, is ApiResult.NetworkError -> emptyList()
                    }
                    _state.value = MovieDetailUiState.Ready(itemResult.value, similar)
                }
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    similarDeferred.cancel()
                    _state.value = MovieDetailUiState.Error
                }
            }
        }
    }

    fun retry() {
        load()
    }

    fun toggleFavorite() {
        val ready = _state.value as? MovieDetailUiState.Ready ?: return
        if (favoriteJob?.isActive == true) {
            return
        }
        val previous = ready.item.favorite
        val next = !previous
        _state.value = ready.copy(item = ready.item.copy(favorite = next))
        favoriteJob = scope.launch {
            val result = safe {
                if (next) setItemList(UserList.Favorite) else clearItemList(UserList.Favorite)
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is MovieDetailUiState.Ready) {
                            current.copy(item = current.item.copy(favorite = previous))
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
        val ready = _state.value as? MovieDetailUiState.Ready ?: return
        if (watchlistJob?.isActive == true) {
            return
        }
        val previous = ready.item.watchlist
        val next = !previous
        _state.value = ready.copy(item = ready.item.copy(watchlist = next))
        watchlistJob = scope.launch {
            val result = safe {
                if (next) setItemList(UserList.Watchlist) else clearItemList(UserList.Watchlist)
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is MovieDetailUiState.Ready) {
                            current.copy(item = current.item.copy(watchlist = previous))
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Watchlist)
                }
            }
        }
    }

    fun toggleWatched() {
        val ready = _state.value as? MovieDetailUiState.Ready ?: return
        if (watchedJob?.isActive == true) {
            return
        }
        val previous = ready.item.watched
        val next = !previous
        _state.value = ready.copy(item = ready.item.copy(watched = next))
        watchedJob = scope.launch {
            val result = safe {
                if (next) setWatched() else clearWatched()
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is MovieDetailUiState.Ready) {
                            current.copy(item = current.item.copy(watched = previous))
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Watched)
                }
            }
        }
    }

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
        favoriteJob?.cancel()
        watchlistJob?.cancel()
        watchedJob?.cancel()
    }
}
