package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.GenresResponse
import io.github.catlinux.cinecat.core.api.Series
import io.github.catlinux.cinecat.core.api.SeriesResponse
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

data class SeriesUiState(
    val paged: PagedListState<Series> = PagedListState(),
    val genres: List<String> = emptyList(),
    val sort: String = "title",
    val genre: String? = null
) {
    val initialLoading: Boolean get() = paged.initialLoading
    val isEmpty: Boolean
        get() = paged.items.isEmpty() && !paged.loading && !paged.error && paged.endReached
}

class SeriesViewModel(
    private val loadPage: suspend (sort: String, genre: String?, offset: Int) -> ApiResult<SeriesResponse>,
    private val loadGenres: suspend () -> ApiResult<GenresResponse>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(repository: LibraryRepository, scope: CoroutineScope) : this(
        loadPage = { sort, genre, offset ->
            repository.series(
                sort = sort,
                genre = genre,
                limit = PagedListState.PAGE_SIZE,
                offset = offset
            )
        },
        loadGenres = { repository.seriesGenres() },
        scope = scope
    )

    private val _state = MutableStateFlow(SeriesUiState())
    val state: StateFlow<SeriesUiState> = _state.asStateFlow()

    private var pageJob: Job? = null
    private var genresJob: Job? = null

    init {
        loadGenresOnce()
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

    fun setGenre(genre: String?) {
        if (_state.value.genre == genre) {
            return
        }
        pageJob?.cancel()
        pageJob = null
        _state.value = _state.value.copy(genre = genre, paged = PagedListState())
        loadMore()
    }

    fun loadMore() {
        val current = _state.value
        val started = current.paged.begin() ?: return
        val offset = started.nextOffset
        val sort = current.sort
        val genre = current.genre
        _state.value = current.copy(paged = started)
        pageJob = scope.launch {
            val result = try {
                loadPage(sort, genre, offset)
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                ApiResult.NetworkError(IOException(error.message, error))
            }
            _state.update { latest ->
                if (latest.sort != sort || latest.genre != genre) {
                    latest
                } else {
                    latest.copy(
                        paged = when (result) {
                            is ApiResult.Success -> latest.paged.success(
                                offset = offset,
                                total = result.value.total,
                                page = result.value.series
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

    private fun loadGenresOnce() {
        genresJob = scope.launch {
            val result: ApiResult<GenresResponse>? = try {
                loadGenres()
            } catch (error: CancellationException) {
                throw error
            } catch (_: Exception) {
                null
            }
            when (result) {
                is ApiResult.Success -> {
                    _state.update {
                        it.copy(genres = result.value.genres.map { genre -> genre.name })
                    }
                }
                is ApiResult.HttpError, is ApiResult.NetworkError, null -> Unit
            }
        }
    }

    override fun onCleared() {
        pageJob?.cancel()
        genresJob?.cancel()
    }
}
