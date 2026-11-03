package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.LibrariesResponse
import io.github.catlinux.cinecat.core.api.Library
import io.github.catlinux.cinecat.core.api.PhotoItem
import io.github.catlinux.cinecat.core.api.PhotosResponse
import java.io.IOException
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class PhotosViewModelTest {
    private data class Request(
        val libraryId: Long,
        val folder: String,
        val sort: PhotoSort,
        val offset: Int
    )

    @Test
    fun oneAccessibleLibraryOpensItsRootAutomatically() = runTest {
        val requests = mutableListOf<Request>()
        val viewModel = PhotosViewModel(
            loadLibraries = {
                ApiResult.Success(
                    LibrariesResponse(
                        listOf(
                            library(4, "photos", "Fotos"),
                            library(5, "video", "Películas")
                        )
                    )
                )
            },
            loadPhotos = { libraryId, folder, sort, offset ->
                requests += Request(libraryId, folder, sort, offset)
                ApiResult.Success(
                    PhotosResponse(
                        folders = listOf("2024"),
                        items = listOf(photo(1)),
                        total = 1
                    )
                )
            },
            scope = this
        )

        advanceUntilIdle()

        val state = viewModel.state.value
        assertEquals(4L, state.selectedLibrary?.id)
        assertFalse(state.showLibraries)
        assertEquals("", state.folder)
        assertEquals(listOf("2024"), state.folders)
        assertEquals(listOf(1L), state.paged.items.map { it.id })
        assertEquals(listOf(Request(4, "", PhotoSort.Date, 0)), requests)
    }

    @Test
    fun severalLibrariesShowTheListAndLoadWhenOneIsSelected() = runTest {
        val requests = mutableListOf<Request>()
        val viewModel = PhotosViewModel(
            loadLibraries = {
                ApiResult.Success(
                    LibrariesResponse(
                        listOf(
                            library(4, "photos", "Casa"),
                            library(5, "photos", "Viajes"),
                            library(6, "music", "Música")
                        )
                    )
                )
            },
            loadPhotos = { libraryId, folder, sort, offset ->
                requests += Request(libraryId, folder, sort, offset)
                ApiResult.Success(PhotosResponse(items = listOf(photo(libraryId)), total = 1))
            },
            scope = this
        )

        advanceUntilIdle()
        assertNull(viewModel.state.value.selectedLibrary)
        assertTrue(viewModel.state.value.showLibraries)
        assertEquals(2, viewModel.state.value.libraries.size)
        assertTrue(requests.isEmpty())

        viewModel.selectLibrary(library(5, "photos", "Viajes"))
        advanceUntilIdle()

        assertEquals(5L, viewModel.state.value.selectedLibrary?.id)
        assertEquals(listOf(Request(5, "", PhotoSort.Date, 0)), requests)
    }

    @Test
    fun paginationSortFolderAndUpKeepTheQueryInSync() = runTest {
        val requests = mutableListOf<Request>()
        val viewModel = PhotosViewModel(
            loadLibraries = {
                ApiResult.Success(LibrariesResponse(listOf(library(4, "photos", "Fotos"))))
            },
            loadPhotos = { libraryId, folder, sort, offset ->
                requests += Request(libraryId, folder, sort, offset)
                ApiResult.Success(
                    PhotosResponse(
                        folders = if (offset == 0) listOf("2024") else emptyList(),
                        items = listOf(photo(offset + 1L), photo(offset + 2L)),
                        total = 100
                    )
                )
            },
            scope = this
        )

        advanceUntilIdle()
        viewModel.loadMore()
        advanceUntilIdle()
        assertEquals(2, requests.size)
        assertEquals(2, requests.last().offset)

        viewModel.setSort(PhotoSort.Name)
        advanceUntilIdle()
        assertEquals(Request(4, "", PhotoSort.Name, 0), requests.last())

        viewModel.openFolder("2024/Verano")
        advanceUntilIdle()
        assertEquals(Request(4, "2024/Verano", PhotoSort.Name, 0), requests.last())

        assertTrue(viewModel.goUp())
        advanceUntilIdle()
        assertEquals("2024", viewModel.state.value.folder)
        assertEquals(Request(4, "2024", PhotoSort.Name, 0), requests.last())

        assertTrue(viewModel.goUp())
        advanceUntilIdle()
        assertEquals("", viewModel.state.value.folder)
        assertFalse(viewModel.goUp())
    }

    @Test
    fun upFromTheRootReturnsToTheLibraryListWithSeveralLibraries() = runTest {
        val viewModel = PhotosViewModel(
            loadLibraries = {
                ApiResult.Success(
                    LibrariesResponse(
                        listOf(
                            library(4, "photos", "Casa"),
                            library(5, "photos", "Viajes")
                        )
                    )
                )
            },
            loadPhotos = { _, _, _, _ ->
                ApiResult.Success(PhotosResponse(items = listOf(photo(1)), total = 1))
            },
            scope = this
        )

        advanceUntilIdle()
        viewModel.selectLibrary(library(4, "photos", "Casa"))
        advanceUntilIdle()
        assertTrue(viewModel.goUp())
        advanceUntilIdle()
        assertNull(viewModel.state.value.selectedLibrary)
        assertTrue(viewModel.state.value.showLibraries)
    }

    @Test
    fun libraryAndPageFailuresCanBeRetried() = runTest {
        var libraryAttempts = 0
        var pageAttempts = 0
        val viewModel = PhotosViewModel(
            loadLibraries = {
                libraryAttempts++
                if (libraryAttempts == 1) {
                    ApiResult.NetworkError(IOException("sin red"))
                } else {
                    ApiResult.Success(LibrariesResponse(listOf(library(4, "photos", "Fotos"))))
                }
            },
            loadPhotos = { _, _, _, _ ->
                pageAttempts++
                if (pageAttempts == 1) {
                    ApiResult.HttpError(500, "fallo")
                } else {
                    ApiResult.Success(PhotosResponse(items = listOf(photo(1)), total = 1))
                }
            },
            scope = this
        )

        advanceUntilIdle()
        assertTrue(viewModel.state.value.librariesError)
        viewModel.retry()
        advanceUntilIdle()
        assertFalse(viewModel.state.value.librariesError)
        assertEquals(4L, viewModel.state.value.selectedLibrary?.id)
        assertTrue(viewModel.state.value.paged.error)

        viewModel.retry()
        advanceUntilIdle()
        assertFalse(viewModel.state.value.paged.error)
        assertEquals(listOf(1L), viewModel.state.value.paged.items.map { it.id })
    }

    @Test
    fun purePhotoHelpersWorkAcrossEdges() {
        assertEquals("2024", parentFolder("2024/Verano"))
        assertEquals("", parentFolder("2024"))
        assertNull(parentFolder(""))
        assertEquals("2024/Verano", childFolder("2024", "Verano"))
        assertEquals("Verano", childFolder("", "Verano"))

        assertEquals(1, nextPhotoIndex(0, 3))
        assertEquals(0, nextPhotoIndex(2, 3))
        assertEquals(2, previousPhotoIndex(0, 3))
        assertEquals(1, previousPhotoIndex(2, 3))

        assertEquals(4f / 3f, photoAspectRatio(4, 3), 0.0001f)
        assertEquals(4f / 3f, photoAspectRatio(0, 0), 0.0001f)
        assertEquals("/api/v1/photos/7/thumb", photoThumbPath(7))
        assertEquals("/api/v1/photos/7/original", photoOriginalPath(7))
        assertTrue(photoDateLabel(1600000000).contains("2020"))
        assertEquals("", photoDateLabel(null))
    }

    private fun library(id: Long, kind: String, name: String): Library =
        Library(id = id, name = name, kind = kind)

    private fun photo(id: Long): PhotoItem =
        PhotoItem(
            id = id,
            name = "IMG_$id.jpg",
            width = 4000,
            height = 3000,
            takenAt = 1600000000,
            size = 1234
        )
}
