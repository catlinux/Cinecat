package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.createApi
import kotlinx.coroutines.test.runTest
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class LibraryRepositoryTest {
    @get:Rule
    val server = MockWebServer()

    @Test
    fun portalOrdersRowsAndSplitsContinueAndListsByKind() = runTest {
        val api = createApi(server.url("/"), OkHttpClient())
        val repository = LibraryRepository(apiProvider = { api })
        val rawRows = """[
            {"id":"favorites","visible":true,"title":"Favoritos","kicker":"Tu selección"},
            {"id":"continue_movies","visible":true,"title":"Seguir películas","kicker":"En pausa"},
            {"id":"continue_series","visible":true,"title":"Seguir series","kicker":"En pausa"},
            {"id":"recent_series","visible":true,"title":"Series nuevas","kicker":"Estrenos"},
            {"id":"watchlist_movies","visible":true,"title":"Películas por ver","kicker":"Después"},
            {"id":"watchlist_series","visible":true,"title":"Series por ver","kicker":"Después"},
            {"id":"next_up","visible":true,"title":"Siguiente","kicker":"Series"},
            {"id":"recent_movies","visible":false,"title":"Oculta","kicker":"No"}
        ]"""
        server.enqueue(jsonResponse(homeJson()))
        val escapedRows = rawRows
            .replace("\\", "\\\\")
            .replace("\"", "\\\"")
            .replace("\n", "\\n")
        server.enqueue(
            jsonResponse(
                """{"ui.show_series":true,"ui.portal_rows":"$escapedRows","unknown":true}"""
            )
        )

        val result = repository.portal()

        assertTrue(result is ApiResult.Success)
        val rows = (result as ApiResult.Success).value.rows
        assertEquals(
            listOf(
                "favorites",
                "continue_movies",
                "continue_series",
                "recent_series",
                "watchlist_movies",
                "watchlist_series"
            ),
            rows.map { it.id }
        )
        assertEquals(listOf(7L), rows[0].mediaItems.map { it.id })
        assertEquals(listOf(8L), rows[0].seriesItems.map { it.id })
        assertEquals(listOf(1L), rows[1].mediaItems.map { it.id })
        assertEquals(listOf(2L), rows[2].mediaItems.map { it.id })
        assertEquals(listOf(4L), rows[3].seriesItems.map { it.id })
        assertEquals(listOf(5L), rows[4].mediaItems.map { it.id })
        assertEquals(listOf(6L), rows[5].seriesItems.map { it.id })
    }

    @Test
    fun portalWithoutServerReturnsNetworkError() = runTest {
        val result = LibraryRepository(apiProvider = { null }).portal()

        assertTrue(result is ApiResult.NetworkError)
    }

    @Test
    fun missingSettingsFallsBackToDefaultRows() = runTest {
        val api = createApi(server.url("/"), OkHttpClient())
        val repository = LibraryRepository(apiProvider = { api })
        server.enqueue(jsonResponse(homeJson()))
        server.enqueue(jsonResponse("""{"ui.show_series":true}"""))

        val result = repository.portal()

        assertTrue(result is ApiResult.Success)
        val ids = (result as ApiResult.Success).value.rows.map { it.id }
        assertTrue(ids.contains("recent_movies"))
        assertTrue(ids.contains("recent_series"))
        assertTrue(ids.contains("favorites"))
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)

    private fun mediaJson(id: Long, title: String): String =
        """{
          "id":$id,
          "library_id":1,
          "library_kind":"video",
          "title":"$title",
          "file_title":"$title",
          "year":2024,
          "kind":"movie",
          "season":null,
          "episode":null,
          "tmdb_id":null,
          "overview":null,
          "poster_url":null,
          "original_title":null,
          "tagline":null,
          "genres":[],
          "runtime_minutes":null,
          "rating":null,
          "directors":[],
          "cast":[],
          "credits":{"cast":[],"crew":[]},
          "backdrop_url":null,
          "watched":false,
          "favorite":false,
          "watchlist":false,
          "progress":null,
          "collections":[],
          "series_id":null,
          "series_title":null,
          "episode_title":null,
          "still_url":null,
          "series_poster_url":null,
          "series_backdrop_url":null
        }""".trimIndent()

    private fun episodeJson(id: Long): String =
        mediaJson(id, "Episodio $id").replace("\"kind\":\"movie\"", "\"kind\":\"episode\"")

    private fun seriesJson(id: Long): String =
        """{
          "id":$id,
          "title":"Serie $id",
          "file_title":"Serie $id",
          "year":2024,
          "poster_url":null,
          "genres":[],
          "season_count":1,
          "episode_count":2,
          "watched_count":0,
          "favorite":false,
          "watchlist":false
        }""".trimIndent()

    private fun homeJson(): String =
        """{
          "continue":[${mediaJson(1, "Pelicula 1")},${episodeJson(2)}],
          "next_up":[],
          "recent_movies":[${mediaJson(3, "Pelicula 3")}],
          "recently_watched":[],
          "recent_series":[${seriesJson(4)}],
          "watchlist":[
            {"kind":"item","item":${mediaJson(5, "Pelicula 5")}},
            {"kind":"series","series":${seriesJson(6)}},
            {"kind":"item"}
          ],
          "favorites":[
            {"kind":"item","item":${mediaJson(7, "Pelicula 7")}},
            {"kind":"series","series":${seriesJson(8)}}
          ],
          "has_clips":false,
          "has_music":false,
          "counts":{"movies":5,"episodes":1,"series":3}
        }""".trimIndent()
}
