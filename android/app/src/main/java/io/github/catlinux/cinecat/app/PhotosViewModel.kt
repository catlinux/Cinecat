package io.github.catlinux.cinecat.app

import androidx.lifecycle.ViewModel
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.LibrariesResponse
import io.github.catlinux.cinecat.core.api.Library
import io.github.catlinux.cinecat.core.api.PhotoItem
import io.github.catlinux.cinecat.core.api.PhotosResponse
import io.github.catlinux.cinecat.core.data.LibraryRepository
import java.io.IOException
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class PhotoSort(val apiValue: String) {
    Date("date"),
    Name("name");

    companion object {
        fun fromApi(value: String?): PhotoSort =
            entries.firstOrNull { it.apiValue == value } ?: Date
    }
}

data class PhotosUiState(
    val libraries: List<Library> = emptyList(),
    val selectedLibrary: Library? = null,
    val folder: String = "",
    val sort: PhotoSort = PhotoSort.Date,
    val folders: List<String> = emptyList(),
    val paged: PagedListState<PhotoItem> = PagedListState(),
    val loadingLibraries: Boolean = true,
    val librariesError: Boolean = false
) {
    val showLibraries: Boolean get() = selectedLibrary == null
    val initialLoading: Boolean
        get() = loadingLibraries || (selectedLibrary != null && paged.initialLoading)
    val isEmpty: Boolean
        get() = selectedLibrary != null &&
            paged.items.isEmpty() &&
            !paged.loading &&
            !paged.error &&
            paged.endReached
    val canGoUp: Boolean
        get() = folder.isNotEmpty() || (selectedLibrary != null && libraries.size > 1)
}

class PhotosViewModel(
    private val loadLibraries: suspend () -> ApiResult<LibrariesResponse>,
    private val loadPhotos: suspend (libraryId: Long, folder: String, sort: PhotoSort, offset: Int) -> ApiResult<PhotosResponse>,
    private val scope: CoroutineScope
) : ViewModel() {
    constructor(repository: LibraryRepository, scope: CoroutineScope) : this(
        loadLibraries = { repository.libraries() },
        loadPhotos = { libraryId, folder, sort, offset ->
            repository.photos(
                libraryId = libraryId,
                folder = folder,
                sort = sort.apiValue,
                limit = PagedListState.PAGE_SIZE,
                offset = offset
            )
        },
        scope = scope
    )

    private val _state = MutableStateFlow(PhotosUiState())
    val state: StateFlow<PhotosUiState> = _state.asStateFlow()

    private var librariesJob: Job? = null
    private var pageJob: Job? = null

    init {
        reloadLibraries()
    }

    fun reloadLibraries() {
        librariesJob?.cancel()
        pageJob?.cancel()
        pageJob = null
        _state.value = PhotosUiState()
        librariesJob = scope.launch {
            val result = try {
                loadLibraries()
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                ApiResult.NetworkError(IOException(error.message, error))
            }
            when (result) {
                is ApiResult.Success -> applyLibraries(result.value.libraries)
                is ApiResult.HttpError, is ApiResult.NetworkError -> {
                    _state.value = _state.value.copy(
                        loadingLibraries = false,
                        librariesError = true
                    )
                }
            }
        }
    }

    private fun applyLibraries(libraries: List<Library>) {
        val photos = libraries.filter { it.kind == "photos" }
        _state.value = _state.value.copy(
            libraries = photos,
            loadingLibraries = false,
            librariesError = false,
            selectedLibrary = null,
            folder = "",
            folders = emptyList(),
            paged = PagedListState()
        )
        if (photos.size == 1) {
            selectLibrary(photos.first())
        }
    }

    fun selectLibrary(library: Library) {
        if (library.kind != "photos") {
            return
        }
        pageJob?.cancel()
        pageJob = null
        _state.value = _state.value.copy(
            selectedLibrary = library,
            folder = "",
            sort = PhotoSort.Date,
            folders = emptyList(),
            paged = PagedListState()
        )
        loadMore()
    }

    fun openFolder(folder: String) {
        val current = _state.value
        if (current.selectedLibrary == null) {
            return
        }
        val clean = folder.trim('/').trim()
        if (clean == current.folder) {
            return
        }
        pageJob?.cancel()
        pageJob = null
        _state.value = current.copy(
            folder = clean,
            folders = emptyList(),
            paged = PagedListState()
        )
        loadMore()
    }

    fun setSort(sort: PhotoSort) {
        val current = _state.value
        if (current.selectedLibrary == null || current.sort == sort) {
            return
        }
        pageJob?.cancel()
        pageJob = null
        _state.value = current.copy(
            sort = sort,
            folders = emptyList(),
            paged = PagedListState()
        )
        loadMore()
    }

    // Devuelve true cuando había un nivel al que volver.
    fun goUp(): Boolean {
        val current = _state.value
        if (current.folder.isNotEmpty()) {
            openFolder(parentFolder(current.folder) ?: "")
            return true
        }
        if (current.selectedLibrary != null && current.libraries.size > 1) {
            pageJob?.cancel()
            pageJob = null
            _state.value = current.copy(
                selectedLibrary = null,
                folder = "",
                folders = emptyList(),
                paged = PagedListState()
            )
            return true
        }
        return false
    }

    fun loadMore() {
        val current = _state.value
        val library = current.selectedLibrary ?: return
        val started = current.paged.begin() ?: return
        val offset = started.nextOffset
        val folder = current.folder
        val sort = current.sort
        _state.value = current.copy(paged = started)
        pageJob = scope.launch {
            val result = try {
                loadPhotos(library.id, folder, sort, offset)
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                ApiResult.NetworkError(IOException(error.message, error))
            }
            _state.update { latest ->
                if (latest.selectedLibrary?.id != library.id ||
                    latest.folder != folder ||
                    latest.sort != sort
                ) {
                    latest
                } else {
                    latest.copy(
                        paged = when (result) {
                            is ApiResult.Success -> latest.paged.success(
                                offset = offset,
                                total = result.value.total,
                                page = result.value.items
                            )
                            is ApiResult.HttpError, is ApiResult.NetworkError -> latest.paged.failure()
                        },
                        folders = if (result is ApiResult.Success && offset == 0) {
                            result.value.folders
                        } else {
                            latest.folders
                        }
                    )
                }
            }
        }
    }

    fun retry() {
        if (_state.value.librariesError) {
            reloadLibraries()
        } else {
            loadMore()
        }
    }

    override fun onCleared() {
        librariesJob?.cancel()
        pageJob?.cancel()
    }
}

fun parentFolder(folder: String): String? {
    if (folder.isBlank()) {
        return null
    }
    return if (folder.contains('/')) {
        folder.substringBeforeLast('/')
    } else {
        ""
    }
}

fun childFolder(parent: String, child: String): String =
    if (parent.isBlank()) child else "$parent/$child"

fun nextPhotoIndex(index: Int, total: Int): Int {
    if (total <= 0 || index < 0) {
        return 0
    }
    if (index >= total) {
        return 0
    }
    return (index + 1) % total
}

fun previousPhotoIndex(index: Int, total: Int): Int {
    if (total <= 0 || index < 0) {
        return 0
    }
    if (index >= total) {
        return total - 1
    }
    return (index - 1 + total) % total
}

fun photoAspectRatio(width: Int, height: Int): Float =
    if (width > 0 && height > 0) width.toFloat() / height.toFloat() else 4f / 3f

fun photoThumbPath(id: Long): String = "/api/v1/photos/$id/thumb"

fun photoOriginalPath(id: Long): String = "/api/v1/photos/$id/original"

private val photoDateFormatter =
    DateTimeFormatter.ofPattern("d MMM yyyy", Locale.forLanguageTag("es-ES"))

fun photoDateLabel(takenAt: Long?): String {
    if (takenAt == null) {
        return ""
    }
    return photoDateFormatter.format(
        Instant.ofEpochSecond(takenAt).atZone(ZoneId.systemDefault())
    )
}
