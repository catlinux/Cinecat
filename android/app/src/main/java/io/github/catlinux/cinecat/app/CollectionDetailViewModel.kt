package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CollectionDetail as ApiCollectionDetail
import io.github.catlinux.cinecat.core.api.UserList
import io.github.catlinux.cinecat.core.data.LibraryRepository
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

sealed interface CollectionDetailUiState {
    data object Loading : CollectionDetailUiState
    data object Error : CollectionDetailUiState
    data class Ready(val collection: ApiCollectionDetail) : CollectionDetailUiState
}

class CollectionDetailViewModel(
    val id: Long,
    private val loadCollection: suspend () -> ApiResult<ApiCollectionDetail>,
    private val setCollectionList: suspend (UserList) -> ApiResult<Unit>,
    private val clearCollectionList: suspend (UserList) -> ApiResult<Unit>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(id: Long, repository: LibraryRepository, scope: CoroutineScope) : this(
        id = id,
        loadCollection = { repository.collection(id) },
        setCollectionList = { list -> repository.setCollectionList(id, list) },
        clearCollectionList = { list -> repository.clearCollectionList(id, list) },
        scope = scope
    )

    private val _state = MutableStateFlow<CollectionDetailUiState>(CollectionDetailUiState.Loading)
    val state: StateFlow<CollectionDetailUiState> = _state.asStateFlow()

    private val eventChannel = Channel<DetailAction>(Channel.BUFFERED)
    val events: Flow<DetailAction> = eventChannel.receiveAsFlow()

    private var loadJob: Job? = null
    private var favoriteJob: Job? = null
    private var watchlistJob: Job? = null

    init {
        load()
    }

    fun load() {
        loadJob?.cancel()
        _state.value = CollectionDetailUiState.Loading
        loadJob = scope.launch {
            val result = safe { loadCollection() }
            _state.value = when (result) {
                is ApiResult.Success -> CollectionDetailUiState.Ready(result.value)
                is ApiResult.HttpError, is ApiResult.NetworkError -> CollectionDetailUiState.Error
            }
        }
    }

    fun retry() {
        load()
    }

    fun toggleFavorite() {
        val ready = _state.value as? CollectionDetailUiState.Ready ?: return
        if (favoriteJob?.isActive == true) {
            return
        }
        val previous = ready.collection.favorite
        val next = !previous
        _state.value = ready.copy(collection = ready.collection.copy(favorite = next))
        favoriteJob = scope.launch {
            val result = safe {
                if (next) {
                    setCollectionList(UserList.Favorite)
                } else {
                    clearCollectionList(UserList.Favorite)
                }
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is CollectionDetailUiState.Ready) {
                            current.copy(
                                collection = current.collection.copy(favorite = previous)
                            )
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
        val ready = _state.value as? CollectionDetailUiState.Ready ?: return
        if (watchlistJob?.isActive == true) {
            return
        }
        val previous = ready.collection.watchlist
        val next = !previous
        _state.value = ready.copy(collection = ready.collection.copy(watchlist = next))
        watchlistJob = scope.launch {
            val result = safe {
                if (next) {
                    setCollectionList(UserList.Watchlist)
                } else {
                    clearCollectionList(UserList.Watchlist)
                }
            }
            when (result) {
                is ApiResult.Success -> Unit
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.update { current ->
                        if (current is CollectionDetailUiState.Ready) {
                            current.copy(
                                collection = current.collection.copy(watchlist = previous)
                            )
                        } else {
                            current
                        }
                    }
                    eventChannel.trySend(DetailAction.Watchlist)
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
    }
}
