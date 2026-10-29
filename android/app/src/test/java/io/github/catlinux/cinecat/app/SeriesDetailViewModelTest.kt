package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.SeasonDetail
import io.github.catlinux.cinecat.core.api.SeriesDetail
import io.github.catlinux.cinecat.core.api.SimilarSeriesResponse
import io.github.catlinux.cinecat.core.api.UserList
import java.io.IOException
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class SeriesDetailViewModelTest {
    @Test
    fun nextEpisodeSelectsItsSeason() = runTest {
        val next = sampleEpisode(id = 20, season = 2, episode = 1)
        val series = sampleSeriesDetail(
            seasons = listOf(sampleSeason(1), sampleSeason(2)),
            nextEpisode = next
        )
        val harness = createHarness(
            series = series,
            seasonDetails = mapOf(
                1 to sampleSeasonDetail(1, emptyList()),
                2 to sampleSeasonDetail(2, emptyList())
            )
        )

        advanceUntilIdle()

        val ready = harness.state.value as SeriesDetailUiState.Ready
        assertEquals(2, ready.selectedSeason)
        assertEquals(listOf(2), harness.seasonRequests)
    }

    @Test
    fun withoutNextUsesFirstNormalSeasonAndSpecialsGoLast() = runTest {
        val series = sampleSeriesDetail(
            seasons = listOf(sampleSeason(0), sampleSeason(2), sampleSeason(1))
        )
        val harness = createHarness(
            series = series,
            seasonDetails = mapOf(
                1 to sampleSeasonDetail(1, emptyList()),
                2 to sampleSeasonDetail(2, emptyList())
            )
        )

        advanceUntilIdle()

        val ready = harness.state.value as SeriesDetailUiState.Ready
        assertEquals(1, ready.selectedSeason)
        assertEquals(listOf(1, 2, 0), ready.seasons.map { it.season })
        assertEquals(listOf(1), harness.seasonRequests)
    }

    @Test
    fun onlySpecialsSelectsZero() = runTest {
        val series = sampleSeriesDetail(seasons = listOf(sampleSeason(0)))
        val harness = createHarness(
            series = series,
            seasonDetails = mapOf(0 to sampleSeasonDetail(0, emptyList()))
        )

        advanceUntilIdle()

        assertEquals(
            0,
            (harness.state.value as SeriesDetailUiState.Ready).selectedSeason
        )
    }

    @Test
    fun changingSeasonLoadsItAndCacheAvoidsASecondRequest() = runTest {
        val series = sampleSeriesDetail(seasons = listOf(sampleSeason(1), sampleSeason(2)))
        val harness = createHarness(
            series = series,
            seasonDetails = mapOf(
                1 to sampleSeasonDetail(1, listOf(sampleEpisode(11, 1, 1))),
                2 to sampleSeasonDetail(2, listOf(sampleEpisode(21, 2, 1)))
            )
        )
        advanceUntilIdle()

        harness.selectSeason(2)
        advanceUntilIdle()
        harness.selectSeason(1)
        advanceUntilIdle()

        assertEquals(listOf(1, 2), harness.seasonRequests)
        assertEquals(1, (harness.state.value as SeriesDetailUiState.Ready).selectedSeason)
    }

    @Test
    fun similarFailureKeepsTheSeriesWithoutTheRow() = runTest {
        val harness = createHarness(
            series = sampleSeriesDetail(seasons = listOf(sampleSeason(1))),
            seasonDetails = mapOf(1 to sampleSeasonDetail(1, emptyList())),
            loadSimilar = { ApiResult.NetworkError(IOException("sin red")) }
        )

        advanceUntilIdle()

        val ready = harness.state.value as SeriesDetailUiState.Ready
        assertTrue(ready.similar.isEmpty())
    }

    @Test
    fun seasonErrorKeepsTheSeriesHeader() = runTest {
        val series = sampleSeriesDetail(seasons = listOf(sampleSeason(1)))
        val harness = createHarness(
            series = series,
            loadSeason = { ApiResult.NetworkError(IOException("sin red")) }
        )

        advanceUntilIdle()

        val ready = harness.state.value as SeriesDetailUiState.Ready
        assertTrue(ready.episodes is SeasonEpisodesState.Error)
        assertEquals("Serie", ready.series.title)
    }

    @Test
    fun favoriteIsOptimisticAndRollsBackOnFailure() = runTest {
        val successHarness = createHarness(
            series = sampleSeriesDetail(seasons = listOf(sampleSeason(1))),
            seasonDetails = mapOf(1 to sampleSeasonDetail(1, emptyList())),
            setSeriesList = { ApiResult.Success(Unit) }
        )
        advanceUntilIdle()

        successHarness.toggleFavorite()
        assertTrue((successHarness.state.value as SeriesDetailUiState.Ready).series.favorite)
        advanceUntilIdle()
        assertEquals(1, successHarness.favoriteSetCalls)

        val failureHarness = createHarness(
            series = sampleSeriesDetail(seasons = listOf(sampleSeason(1))),
            seasonDetails = mapOf(1 to sampleSeasonDetail(1, emptyList())),
            setSeriesList = { ApiResult.NetworkError(IOException("sin red")) }
        )
        advanceUntilIdle()

        failureHarness.toggleFavorite()
        advanceUntilIdle()

        assertFalse((failureHarness.state.value as SeriesDetailUiState.Ready).series.favorite)
        assertEquals(DetailAction.Favorite, failureHarness.events.first())
    }

    @Test
    fun seasonWatchedCallsEndpointAndReloadsSeason() = runTest {
        val series = sampleSeriesDetail(seasons = listOf(sampleSeason(1)))
        val harness = createHarness(
            series = series,
            seasonDetails = mapOf(1 to sampleSeasonDetail(1, listOf(sampleEpisode(11, 1, 1)))),
            setSeasonWatched = { ApiResult.Success(Unit) }
        )
        advanceUntilIdle()

        harness.toggleSeasonWatched()
        advanceUntilIdle()

        assertEquals(1, harness.setSeasonWatchedCalls)
        assertEquals(2, harness.seasonRequests.size)
        assertTrue(
            (harness.state.value as SeriesDetailUiState.Ready).episodes is
                SeasonEpisodesState.Ready
        )
    }

    @Test
    fun episodeWatchedUsesPutAndRollsBackOnFailure() = runTest {
        val episode = sampleEpisode(id = 11, season = 1, episode = 1)
        val series = sampleSeriesDetail(seasons = listOf(sampleSeason(1)))
        val successHarness = createHarness(
            series = series,
            seasonDetails = mapOf(1 to sampleSeasonDetail(1, listOf(episode))),
            setEpisodeWatched = { ApiResult.Success(Unit) }
        )
        advanceUntilIdle()

        successHarness.toggleEpisodeWatched(episode.id)
        advanceUntilIdle()
        assertTrue(
            (successHarness.state.value as SeriesDetailUiState.Ready)
                .episodeItems.single().watched
        )
        assertEquals(1, successHarness.setEpisodeWatchedCalls)

        val failureHarness = createHarness(
            series = series,
            seasonDetails = mapOf(1 to sampleSeasonDetail(1, listOf(episode))),
            setEpisodeWatched = { ApiResult.NetworkError(IOException("sin red")) }
        )
        advanceUntilIdle()

        failureHarness.toggleEpisodeWatched(episode.id)
        advanceUntilIdle()

        assertFalse(
            (failureHarness.state.value as SeriesDetailUiState.Ready)
                .episodeItems.single().watched
        )
    }

    private fun TestScope.createHarness(
        series: SeriesDetail,
        seasonDetails: Map<Int, SeasonDetail> = emptyMap(),
        loadSeason: suspend (Int) -> ApiResult<SeasonDetail> = { season ->
            seasonDetails[season]
                ?.let { ApiResult.Success(it) }
                ?: ApiResult.NetworkError(IOException("sin temporada"))
        },
        setSeriesList: suspend (UserList) -> ApiResult<Unit> = { ApiResult.Success(Unit) },
        setSeasonWatched: suspend (Int) -> ApiResult<Unit> = { ApiResult.Success(Unit) },
        setEpisodeWatched: suspend (Long) -> ApiResult<Unit> = { ApiResult.Success(Unit) },
        loadSimilar: suspend () -> ApiResult<SimilarSeriesResponse> = {
            ApiResult.Success(SimilarSeriesResponse(emptyList()))
        }
    ): TestHarness {
        val counters = Counters()
        val viewModel = SeriesDetailViewModel(
            id = series.id,
            loadSeries = { ApiResult.Success(series) },
            loadSeason = { season ->
                counters.seasonRequests += season
                loadSeason(season)
            },
            loadSimilar = loadSimilar,
            setSeriesList = { list ->
                counters.favoriteSetCalls++
                setSeriesList(list)
            },
            clearSeriesList = { ApiResult.Success(Unit) },
            setSeriesWatched = { ApiResult.Success(Unit) },
            clearSeriesWatched = { ApiResult.Success(Unit) },
            setSeasonWatched = { season ->
                counters.setSeasonWatchedCalls++
                setSeasonWatched(season)
            },
            clearSeasonWatched = { ApiResult.Success(Unit) },
            setEpisodeWatched = { id ->
                counters.setEpisodeWatchedCalls++
                setEpisodeWatched(id)
            },
            clearEpisodeWatched = { ApiResult.Success(Unit) },
            scope = this
        )
        return TestHarness(viewModel, counters)
    }

    private class Counters {
        val seasonRequests = mutableListOf<Int>()
        var favoriteSetCalls = 0
        var setSeasonWatchedCalls = 0
        var setEpisodeWatchedCalls = 0
    }

    private class TestHarness(
        val viewModel: SeriesDetailViewModel,
        private val counters: Counters
    ) {
        val state get() = viewModel.state
        val events get() = viewModel.events
        val seasonRequests get() = counters.seasonRequests
        val favoriteSetCalls get() = counters.favoriteSetCalls
        val setSeasonWatchedCalls get() = counters.setSeasonWatchedCalls
        val setEpisodeWatchedCalls get() = counters.setEpisodeWatchedCalls

        fun selectSeason(season: Int) = viewModel.selectSeason(season)
        fun toggleFavorite() = viewModel.toggleFavorite()
        fun toggleSeasonWatched() = viewModel.toggleSeasonWatched()
        fun toggleEpisodeWatched(id: Long) = viewModel.toggleEpisodeWatched(id)
    }
}
