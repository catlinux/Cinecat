package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.GenresResponse
import io.github.catlinux.cinecat.core.api.ItemsResponse
import io.github.catlinux.cinecat.core.api.LetterIndex
import io.github.catlinux.cinecat.core.api.MediaItem
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

data class MoviesUiState(
    val paged: PagedListState<MediaItem> = PagedListState(),
    val genres: List<String> = emptyList(),
    val sort: String = "title",
    val genre: String? = null,
    val letters: List<LetterIndex> = emptyList()
) {
    val initialLoading: Boolean get() = paged.initialLoading
    val isEmpty: Boolean
        get() = paged.items.isEmpty() && !paged.loading && !paged.error && paged.endReached
}

class MoviesViewModel(
    private val loadPage: suspend (sort: String, genre: String?, offset: Int, letters: Boolean) -> ApiResult<ItemsResponse>,
    private val loadGenres: suspend () -> ApiResult<GenresResponse>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(repository: LibraryRepository, scope: CoroutineScope) : this(
        loadPage = { sort, genre, offset, letters ->
            repository.items(
                kind = "movie",
                sort = sort,
                genre = genre,
                limit = PagedListState.PAGE_SIZE,
                offset = offset,
                letters = letters
            )
        },
        loadGenres = { repository.genres("movie") },
        scope = scope
    )

    private val _state = MutableStateFlow(MoviesUiState())
    val state: StateFlow<MoviesUiState> = _state.asStateFlow()

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
        _state.value = _state.value.copy(
            sort = sort,
            paged = PagedListState(),
            letters = emptyList()
        )
        loadMore()
    }

    fun setGenre(genre: String?) {
        if (_state.value.genre == genre) {
            return
        }
        pageJob?.cancel()
        pageJob = null
        _state.value = _state.value.copy(
            genre = genre,
            paged = PagedListState(),
            letters = emptyList()
        )
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
                loadPage(sort, genre, offset, offset == 0 && sort == "title")
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
                                page = result.value.items
                            )
                            is ApiResult.HttpError -> latest.paged.failure()
                            is ApiResult.NetworkError -> latest.paged.failure()
                        },
                        letters = if (result is ApiResult.Success && offset == 0) {
                            result.value.letters
                        } else {
                            latest.letters
                        }
                    )
                }
            }
        }
    }

    fun retry() {
        loadMore()
    }

    suspend fun ensureLoaded(index: Int) {
        while (_state.value.paged.items.size <= index) {
            val before = _state.value.paged
            if (before.error || before.endReached) {
                return
            }
            loadMore()
            val job = pageJob ?: return
            job.join()
            val after = _state.value.paged
            if (after.error || after.endReached || after.items.size <= before.items.size) {
                return
            }
        }
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
