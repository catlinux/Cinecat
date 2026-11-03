package io.github.catlinux.cinecat.core.api

import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class PhotosApiTest {
    @get:Rule
    val server = MockWebServer()

    private fun api(): CinecatApi = createApi(server.url("/"), OkHttpClient())

    @Test
    fun parsesVisibleLibrariesAndPhotos() = runTest {
        server.enqueue(
            jsonResponse(
                """{"libraries":[{"id":4,"name":"Fotos","kind":"photos","item_count":3,"paths":[]}]}"""
            )
        )
        val libraries = api().libraries()
        assertTrue(libraries is ApiResult.Success)
        val library = (libraries as ApiResult.Success).value.libraries.single()
        assertEquals(4L, library.id)
        assertEquals("Fotos", library.name)
        assertEquals("photos", library.kind)
        assertEquals(3L, library.itemCount)
        assertEquals("/api/v1/libraries", server.takeRequest().path)

        server.enqueue(
            jsonResponse(
                """{
                  "folders":["2024"],
                  "items":[
                    {"id":7,"name":"IMG_7.jpg","width":4000,"height":3000,"taken_at":1600000000,"size":1234},
                    {"id":8,"name":"IMG_8.png","width":3000,"height":4000,"taken_at":null,"size":5678}
                  ],
                  "total":2,
                  "desconocido":true
                }"""
            )
        )
        val photos = api().photos(
            libraryId = 4,
            folder = "2024/Verano",
            sort = "name",
            limit = 60,
            offset = 120
        )
        assertTrue(photos is ApiResult.Success)
        val value = (photos as ApiResult.Success).value
        assertEquals(listOf("2024"), value.folders)
        assertEquals(2, value.items.size)
        assertEquals(7L, value.items.first().id)
        assertEquals(4000, value.items.first().width)
        assertEquals(1600000000L, value.items.first().takenAt)
        assertNull(value.items[1].takenAt)
        assertEquals(2L, value.total)

        val request = server.takeRequest()
        assertEquals("/api/v1/photos", request.requestUrl?.encodedPath)
        assertEquals("4", request.requestUrl?.queryParameter("library_id"))
        assertEquals("2024/Verano", request.requestUrl?.queryParameter("folder"))
        assertEquals("name", request.requestUrl?.queryParameter("sort"))
        assertEquals("60", request.requestUrl?.queryParameter("limit"))
        assertEquals("120", request.requestUrl?.queryParameter("offset"))
    }

    @Test
    fun photosWithoutOptionalQueryParametersOnlySendLibrary() = runTest {
        server.enqueue(jsonResponse("""{"folders":[],"items":[],"total":0}"""))
        val result = api().photos(libraryId = 9)
        assertTrue(result is ApiResult.Success)
        val request = server.takeRequest()
        assertEquals("/api/v1/photos", request.requestUrl?.encodedPath)
        assertEquals("9", request.requestUrl?.queryParameter("library_id"))
        assertNull(request.requestUrl?.queryParameter("folder"))
        assertNull(request.requestUrl?.queryParameter("sort"))
        assertNull(request.requestUrl?.queryParameter("limit"))
        assertNull(request.requestUrl?.queryParameter("offset"))
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)
}
