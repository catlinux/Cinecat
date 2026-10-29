package io.github.catlinux.cinecat.core.api

import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonPrimitive
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class LibraryApiTest {
    @get:Rule
    val server = MockWebServer()

    private fun api(): CinecatApi = createApi(server.url("/"), OkHttpClient())

    @Test
    fun toleratesOmittedFieldsAndNullListsFromTheServer() = runTest {
        // Los listados del servidor omiten cast y credits, y Go envía null en slices vacíos.
        server.enqueue(
            jsonResponse(
                """{"continue":[{"id":7,"title":"Sin reparto","genres":null}],"next_up":null,"counts":{"movies":1}}"""
            )
        )
        val home = api().home()
        assertTrue(home is ApiResult.Success)
        val value = (home as ApiResult.Success).value
        val item = value.continueWatching.single()
        assertEquals("Sin reparto", item.title)
        assertTrue(item.genres.isEmpty())
        assertTrue(item.credits.cast.isEmpty())
        assertTrue(value.nextUp.isEmpty())
        assertEquals(1, value.counts.movies)
    }

    @Test
    fun parsesLibraryReadsAndIgnoresUnknownFields() = runTest {
        server.enqueue(jsonResponse(homeJson()))
        val home = api().home()
        assertTrue(home is ApiResult.Success)
        val homeValue = (home as ApiResult.Success).value
        assertEquals("Pelicula 1", homeValue.continueWatching.first().title)
        assertEquals(1, homeValue.recentSeries.first().id)
        assertEquals("/api/v1/home", server.takeRequest().path)

        val rawRows = """[{"id":"recent_movies","visible":true,"title":"Recientes","kicker":"Ahora"}]"""
        server.enqueue(
            jsonResponse(
                """{"ui.show_series":false,"ui.portal_rows":${JsonPrimitive(rawRows)},"unknown":true}"""
            )
        )
        val settings = api().publicSettings()
        assertTrue(settings is ApiResult.Success)
        val settingsValue = (settings as ApiResult.Success).value
        assertEquals(false, settingsValue.showSeries)
        assertEquals(rawRows, settingsValue.portalRows)
        assertEquals("/api/v1/settings/public", server.takeRequest().path)

        server.enqueue(jsonResponse(itemsJson()))
        val items = api().items()
        assertTrue(items is ApiResult.Success)
        assertEquals("Pelicula 1", (items as ApiResult.Success).value.items.first().title)
        assertEquals("/api/v1/items", server.takeRequest().path)

        server.enqueue(jsonResponse(mediaJson(7, "Pelicula 7")))
        val item = api().item(7)
        assertTrue(item is ApiResult.Success)
        assertEquals(7L, (item as ApiResult.Success).value.id)
        assertEquals("/api/v1/items/7", server.takeRequest().path)

        server.enqueue(jsonResponse("""{"items":[${mediaJson(8, "Pelicula 8")}]}"""))
        val similar = api().itemSimilar(7)
        assertTrue(similar is ApiResult.Success)
        assertEquals("/api/v1/items/7/similar", server.takeRequest().path)

        server.enqueue(jsonResponse(seriesResponseJson()))
        val series = api().series(sort = "title")
        assertTrue(series is ApiResult.Success)
        assertEquals("Serie 1", (series as ApiResult.Success).value.series.first().title)
        assertEquals("/api/v1/series?sort=title", server.takeRequest().path)

        server.enqueue(jsonResponse(seriesDetailJson()))
        val detail = api().seriesDetail(3)
        assertTrue(detail is ApiResult.Success)
        assertEquals(1, (detail as ApiResult.Success).value.seasons.first().season)
        assertEquals("/api/v1/series/3", server.takeRequest().path)

        server.enqueue(jsonResponse(seasonDetailJson()))
        val season = api().season(3, 1)
        assertTrue(season is ApiResult.Success)
        assertEquals(1, (season as ApiResult.Success).value.items.size)
        assertEquals("/api/v1/series/3/seasons/1", server.takeRequest().path)

        server.enqueue(jsonResponse("""{"series":[${seriesJson(4, "Serie 4")}]}"""))
        val similarSeries = api().seriesSimilar(3, 5)
        assertTrue(similarSeries is ApiResult.Success)
        assertEquals("/api/v1/series/3/similar?limit=5", server.takeRequest().path)

        server.enqueue(jsonResponse("""{"genres":[{"name":"Drama","count":2}]}"""))
        val seriesGenres = api().seriesGenres()
        assertTrue(seriesGenres is ApiResult.Success)
        assertEquals("/api/v1/series/genres", server.takeRequest().path)

        server.enqueue(jsonResponse("""{"genres":[{"name":"Drama","count":2}]}"""))
        val genres = api().genres()
        assertTrue(genres is ApiResult.Success)
        assertEquals("/api/v1/genres?kind=movie", server.takeRequest().path)

        server.enqueue(jsonResponse(collectionsJson()))
        val collections = api().collections(limit = 10)
        assertTrue(collections is ApiResult.Success)
        assertEquals("/api/v1/collections?limit=10", server.takeRequest().path)

        server.enqueue(jsonResponse(collectionDetailJson()))
        val collection = api().collection(4)
        assertTrue(collection is ApiResult.Success)
        assertEquals(1, (collection as ApiResult.Success).value.items.size)
        assertEquals("/api/v1/collections/4", server.takeRequest().path)

        server.enqueue(jsonResponse(listsJson()))
        val lists = api().lists(UserList.Favorite, "items")
        assertTrue(lists is ApiResult.Success)
        assertEquals("/api/v1/lists/favorite?kind=items", server.takeRequest().path)
    }

    @Test
    fun nullQueryParametersAreNotSent() = runTest {
        server.enqueue(jsonResponse(itemsJson()))
        api().items(watched = null, limit = null)
        assertEquals("/api/v1/items", server.takeRequest().path)

        server.enqueue(jsonResponse(itemsJson()))
        api().items(
            kind = "movie",
            q = "cine",
            letter = "C",
            genre = "Drama",
            sort = "title",
            watched = true,
            inProgress = false,
            collectionId = 3,
            seriesId = 4,
            limit = 10,
            offset = 5
        )
        val path = server.takeRequest().path.orEmpty()
        assertTrue(path.contains("kind=movie"))
        assertTrue(path.contains("q=cine"))
        assertTrue(path.contains("letter=C"))
        assertTrue(path.contains("genre=Drama"))
        assertTrue(path.contains("sort=title"))
        assertTrue(path.contains("watched=true"))
        assertTrue(path.contains("in_progress=false"))
        assertTrue(path.contains("collection_id=3"))
        assertTrue(path.contains("series_id=4"))
        assertTrue(path.contains("limit=10"))
        assertTrue(path.contains("offset=5"))
    }

    @Test
    fun listAndWatchedActionsUseTheApiPaths() = runTest {
        val api = api()

        enqueueNoContent()
        assertTrue(api.setItemList(7, UserList.Favorite) is ApiResult.Success)
        server.takeRequest().let { request ->
            assertEquals("PUT", request.method)
            assertEquals("/api/v1/items/7/lists/favorite", request.path)
        }

        enqueueNoContent()
        assertTrue(api.clearItemList(7, UserList.Watchlist) is ApiResult.Success)
        server.takeRequest().let { request ->
            assertEquals("DELETE", request.method)
            assertEquals("/api/v1/items/7/lists/watchlist", request.path)
        }

        enqueueNoContent()
        assertTrue(api.setSeriesList(8, UserList.Watchlist) is ApiResult.Success)
        assertEquals("/api/v1/series/8/lists/watchlist", server.takeRequest().path)

        enqueueNoContent()
        assertTrue(api.clearSeriesList(8, UserList.Favorite) is ApiResult.Success)
        assertEquals("/api/v1/series/8/lists/favorite", server.takeRequest().path)

        enqueueNoContent()
        assertTrue(api.setCollectionList(9, UserList.Favorite) is ApiResult.Success)
        assertEquals("/api/v1/collections/9/lists/favorite", server.takeRequest().path)

        enqueueNoContent()
        assertTrue(api.clearCollectionList(9, UserList.Watchlist) is ApiResult.Success)
        assertEquals("/api/v1/collections/9/lists/watchlist", server.takeRequest().path)

        enqueueNoContent()
        assertTrue(api.setWatched(7) is ApiResult.Success)
        server.takeRequest().let { request ->
            assertEquals("PUT", request.method)
            assertEquals("/api/v1/items/7/watched", request.path)
        }

        enqueueNoContent()
        assertTrue(api.clearWatched(7) is ApiResult.Success)
        server.takeRequest().let { request ->
            assertEquals("DELETE", request.method)
            assertEquals("/api/v1/items/7/watched", request.path)
        }

        enqueueNoContent()
        assertTrue(api.setSeriesWatched(8) is ApiResult.Success)
        assertEquals("/api/v1/series/8/watched", server.takeRequest().path)

        enqueueNoContent()
        assertTrue(api.clearSeriesWatched(8) is ApiResult.Success)
        assertEquals("/api/v1/series/8/watched", server.takeRequest().path)

        enqueueNoContent()
        assertTrue(api.setSeasonWatched(8, 2) is ApiResult.Success)
        assertEquals("/api/v1/series/8/seasons/2/watched", server.takeRequest().path)

        enqueueNoContent()
        assertTrue(api.clearSeasonWatched(8, 2) is ApiResult.Success)
        assertEquals("/api/v1/series/8/seasons/2/watched", server.takeRequest().path)
    }

    @Test
    fun tmdbImageOnlyRewritesTmdbSizes() {
        assertEquals(
            "/api/v1/images/tmdb/w342/poster.jpg",
            tmdbImage("/api/v1/images/tmdb/w500/poster.jpg", "w342")
        )
        assertEquals(
            "https://image.tmdb.org/t/p/w342/poster.jpg",
            tmdbImage("https://image.tmdb.org/t/p/w500/poster.jpg", "w342")
        )
        assertEquals(
            "https://image.tmdb.org/t/p/w780/poster.jpg",
            tmdbImage("https://image.tmdb.org/t/p/original/poster.jpg", "w780")
        )
        assertEquals(
            "https://example.com/w500/poster.jpg",
            tmdbImage("https://example.com/w500/poster.jpg", "w342")
        )
        assertEquals(
            "https://image.tmdb.org/t/p/w500",
            tmdbImage("https://image.tmdb.org/t/p/w500", "w342")
        )
        assertNull(tmdbImage(null, "w342"))
    }

    @Test
    fun portalRowsUsesValidJsonAndFallsBackToTheServerOrder() {
        val raw = """[{"id":"favorites","visible":false,"title":"Favoritos","kicker":"Tus">}]"""
        val rows = portalRows(PublicSettings(portalRows = raw))
        assertFalse(rows.isEmpty())
        assertEquals(DefaultPortalRows.map { it.id }, rows.map { it.id })

        val valid = """[{"id":"favorites","visible":true,"title":"Mi lista","kicker":"Yo"}]"""
        val parsed = portalRows(PublicSettings(portalRows = valid))
        assertEquals(1, parsed.size)
        assertEquals("favorites", parsed.single().id)
        assertEquals("Mi lista", parsed.single().title)

        assertEquals(DefaultPortalRows, portalRows(PublicSettings(portalRows = null)))
        assertEquals(DefaultPortalRows, portalRows(PublicSettings(portalRows = "not json")))
        assertEquals(DefaultPortalRows, portalRows(PublicSettings(portalRows = "[]")))
    }

    private fun enqueueNoContent() {
        server.enqueue(MockResponse().setResponseCode(204))
    }

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)

    private fun mediaJson(id: Long, title: String, kind: String = "movie"): String =
        """{
          "id":$id,
          "library_id":1,
          "library_kind":"video",
          "title":"$title",
          "file_title":"$title",
          "year":2024,
          "kind":"$kind",
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
          "series_backdrop_url":null,
          "unknown":"ignored"
        }""".trimIndent()

    private fun seriesJson(id: Long, title: String): String =
        """{
          "id":$id,
          "title":"$title",
          "file_title":"$title",
          "year":2024,
          "poster_url":null,
          "genres":[],
          "season_count":1,
          "episode_count":2,
          "watched_count":0,
          "match_status":null,
          "favorite":false,
          "watchlist":false,
          "unknown":"ignored"
        }""".trimIndent()

    private fun homeJson(): String =
        """{
          "continue":[${mediaJson(1, "Pelicula 1")}],
          "next_up":[],
          "recent_movies":[${mediaJson(2, "Pelicula 2")}],
          "recently_watched":[],
          "recent_series":[${seriesJson(1, "Serie 1")}],
          "watchlist":[],
          "favorites":[],
          "has_clips":false,
          "has_music":false,
          "counts":{"movies":2,"episodes":0,"series":1},
          "unknown":"ignored"
        }""".trimIndent()

    private fun itemsJson(): String =
        """{"items":[${mediaJson(1, "Pelicula 1")}],"total":1,"limit":50,"offset":0}""".trimIndent()

    private fun seriesResponseJson(): String =
        """{"series":[${seriesJson(1, "Serie 1")}],"total":1,"limit":48,"offset":0}"""

    private fun seasonJson(): String =
        """{"season":1,"name":"Temporada 1","overview":null,"poster_url":null,"episode_count":1,"watched_count":0}"""

    private fun seriesDetailJson(): String =
        """{
          "id":3,
          "title":"Serie 3",
          "file_title":"Serie 3",
          "folder":"Serie 3",
          "original_title":null,
          "year":2024,
          "tagline":null,
          "overview":null,
          "overview_language":null,
          "poster_url":null,
          "backdrop_url":null,
          "genres":[],
          "rating":null,
          "directors":[],
          "cast":[],
          "credits":{"cast":[],"crew":[]},
          "tmdb_id":null,
          "match_status":null,
          "match_score":null,
          "poster_custom":false,
          "backdrop_custom":false,
          "season_count":1,
          "episode_count":1,
          "watched_count":0,
          "favorite":false,
          "watchlist":false,
          "seasons":[${seasonJson()}],
          "next_episode":null,
          "unknown":"ignored"
        }""".trimIndent()

    private fun seasonDetailJson(): String =
        """{"season":1,"name":"Temporada 1","overview":null,"poster_url":null,"items":[${mediaJson(1, "Episodio 1", "episode")}]}"""

    private fun collectionJson(): String =
        """{"id":4,"name":"Saga","overview":null,"poster_url":null,"backdrop_url":null,"item_count":1,"source":"manual","edited":false,"favorite":false,"watchlist":false}"""

    private fun collectionsJson(): String =
        """{"collections":[${collectionJson()}],"total":1,"limit":48,"offset":0}"""

    private fun collectionDetailJson(): String =
        """{"id":4,"name":"Saga","overview":null,"poster_url":null,"backdrop_url":null,"item_count":1,"source":"manual","edited":false,"favorite":false,"watchlist":false,"items":[${mediaJson(1, "Pelicula 1")}]}"""

    private fun listsJson(): String =
        """{"items":[${mediaJson(1, "Pelicula 1")}],"series":[],"collections":[],"totals":{"items":1,"series":0,"collections":0},"limit":50,"offset":0}"""
}
