package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.Collection
import io.github.catlinux.cinecat.core.api.CollectionsResponse
import io.github.catlinux.cinecat.core.data.LibraryRepository
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class CollectionsUiState(
    val paged: PagedListState<Collection> = PagedListState(),
    val sort: String = "name"
) {
    val initialLoading: Boolean get() = paged.initialLoading
    val isEmpty: Boolean
        get() = paged.items.isEmpty() && !paged.loading && !paged.error && paged.endReached
}

class CollectionsViewModel(
    private val loadPage: suspend (sort: String, offset: Int) -> ApiResult<CollectionsResponse>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(repository: LibraryRepository, scope: CoroutineScope) : this(
        loadPage = { sort, offset ->
            repository.collections(
                sort = sort,
                limit = PagedListState.PAGE_SIZE,
                offset = offset
            )
        },
        scope = scope
    )

    private val _state = MutableStateFlow(CollectionsUiState())
    val state: StateFlow<CollectionsUiState> = _state.asStateFlow()

    private var pageJob: Job? = null

    init {
        loadMore()
    }

    fun setSort(sort: String) {
        if (_state.value.sort == sort) {
            return
        }
        pageJob?.cancel()
        pageJob = null
        _state.value = _state.value.copy(sort = sort, paged = PagedListState())
        loadMore()
    }

    fun loadMore() {
        val current = _state.value
        val started = current.paged.begin() ?: return
        val offset = started.nextOffset
        val sort = current.sort
        _state.value = current.copy(paged = started)
        pageJob = scope.launch {
            val result = try {
                loadPage(sort, offset)
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                ApiResult.NetworkError(IOException(error.message, error))
            }
            _state.update { latest ->
                if (latest.sort != sort) {
                    latest
                } else {
                    latest.copy(
                        paged = when (result) {
                            is ApiResult.Success -> latest.paged.success(
                                offset = offset,
                                total = result.value.total,
                                page = result.value.collections
                            )
                            is ApiResult.HttpError -> latest.paged.failure()
                            is ApiResult.NetworkError -> latest.paged.failure()
                        }
                    )
                }
            }
        }
    }

    fun retry() {
        loadMore()
    }

    override fun onCleared() {
        pageJob?.cancel()
    }
}
