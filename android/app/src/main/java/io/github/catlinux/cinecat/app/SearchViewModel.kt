package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.ItemsResponse
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.Series
import io.github.catlinux.cinecat.core.api.SeriesResponse
import io.github.catlinux.cinecat.core.data.LibraryRepository
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class SearchUiState(
    val query: String = "",
    val movies: List<MediaItem> = emptyList(),
    val series: List<Series> = emptyList(),
    val loading: Boolean = false,
    val error: Boolean = false,
    val hasSearched: Boolean = false,
    val resultQuery: String = ""
) {
    val isEmpty: Boolean
        get() = hasSearched && !error && movies.isEmpty() && series.isEmpty()
}

class SearchViewModel(
    private val loadMovies: suspend (String) -> ApiResult<ItemsResponse>,
    private val loadSeries: suspend (String) -> ApiResult<SeriesResponse>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(repository: LibraryRepository, scope: CoroutineScope) : this(
        loadMovies = { query -> repository.items(kind = "movie", q = query, limit = 30) },
        loadSeries = { query -> repository.series(q = query, limit = 30) },
        scope = scope
    )

    private val _state = MutableStateFlow(SearchUiState())
    val state: StateFlow<SearchUiState> = _state.asStateFlow()

    private var debounceJob: Job? = null
    private var searchJob: Job? = null

    fun onQueryChanged(value: String) {
        debounceJob?.cancel()
        searchJob?.cancel()
        val query = value.trim()
        _state.value = SearchUiState(query = value)
        if (query.length < 2) {
            return
        }
        debounceJob = scope.launch {
            delay(DEBOUNCE_MS)
            search(query)
        }
    }

    fun searchNow() {
        debounceJob?.cancel()
        val query = _state.value.query.trim()
        if (query.length >= 2) {
            search(query)
        }
    }

    fun retry() {
        val query = _state.value.query.trim()
        if (query.length >= 2) {
            search(query)
        }
    }

    private fun search(query: String) {
        searchJob?.cancel()
        _state.update { it.copy(loading = true, error = false, hasSearched = false) }
        searchJob = scope.launch {
            val result = try {
                coroutineScope {
                    val movies = async { loadMovies(query) }
                    val series = async { loadSeries(query) }
                    movies.await() to series.await()
                }
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                ApiResult.NetworkError(IOException(error.message, error)) to
                    ApiResult.NetworkError(IOException(error.message, error))
            }
            if (_state.value.query.trim() != query) {
                return@launch
            }
            val moviesSuccess = result.first as? ApiResult.Success<*>
            val seriesSuccess = result.second as? ApiResult.Success<*>
            if (moviesSuccess != null && seriesSuccess != null) {
                val movies = moviesSuccess.value as ItemsResponse
                val series = seriesSuccess.value as SeriesResponse
                _state.update {
                    it.copy(
                        movies = movies.items,
                        series = series.series,
                        loading = false,
                        error = false,
                        hasSearched = true,
                        resultQuery = query
                    )
                }
            } else {
                _state.update { it.searchFailed(query) }
            }
        }
    }

    override fun onCleared() {
        debounceJob?.cancel()
        searchJob?.cancel()
    }

    private fun SearchUiState.searchFailed(query: String): SearchUiState = copy(
        movies = emptyList(),
        series = emptyList(),
        loading = false,
        error = true,
        hasSearched = true,
        resultQuery = query
    )

    private companion object {
        const val DEBOUNCE_MS = 300L
    }
}
