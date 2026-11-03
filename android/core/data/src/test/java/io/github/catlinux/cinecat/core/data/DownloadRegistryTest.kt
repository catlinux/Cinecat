package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.DownloadRecord
import io.github.catlinux.cinecat.core.api.DownloadRegistry
import io.github.catlinux.cinecat.core.api.PendingProgressSync
import java.nio.file.Files
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DownloadRegistryTest {
    private val movie = DownloadRecord(
        itemId = 1,
        kind = "movie",
        title = "Película",
        downloadId = 11,
        fileName = "1.mkv",
        sizeBytes = 1024,
        posterUrl = "/poster.jpg"
    )

    private val episode = DownloadRecord(
        itemId = 2,
        kind = "episode",
        title = "Serie · T1E2 · Episodio",
        seriesTitle = "Serie",
        season = 1,
        episode = 2,
        downloadId = 12,
        fileName = "2.mp4",
        sizeBytes = 2048
    )

    @Test
    fun guardaCargaYSerializaElRegistro() = runTest {
        val file = Files.createTempDirectory("cinecat-downloads")
            .resolve("downloads.json")
            .toFile()
        val registry = DownloadRegistry(file)

        registry.add(movie)
        registry.add(episode)
        registry.updateSize(episode.itemId, 4096)
        registry.recordProgress(movie.itemId, 120.0, 3600.0)

        val reloaded = DownloadRegistry(file).load()
        assertEquals(listOf(1L, 2L), reloaded.downloads.map { it.itemId })
        assertEquals(4096L, reloaded.downloads.single { it.itemId == 2L }.sizeBytes)
        assertEquals(120.0, reloaded.downloads.single { it.itemId == 1L }.positionSeconds, 0.0)
        assertEquals(1, reloaded.pendingProgress.size)
        assertEquals(120.0, reloaded.pendingProgress.single().positionSeconds, 0.0)
    }

    @Test
    fun borraElRegistroYSuProgresoPendiente() = runTest {
        val file = Files.createTempDirectory("cinecat-downloads")
            .resolve("downloads.json")
            .toFile()
        val registry = DownloadRegistry(file)
        registry.add(movie)
        registry.add(episode)
        registry.recordProgress(movie.itemId, 10.0, 100.0)
        registry.recordProgress(episode.itemId, 20.0, 200.0)

        registry.remove(movie.itemId)

        assertEquals(listOf(2L), registry.load().downloads.map { it.itemId })
        assertEquals(listOf(2L), registry.load().pendingProgress.map { it.itemId })
    }

    @Test
    fun vaciaSoloLosPendientesQueSeEnvian() = runTest {
        val file = Files.createTempDirectory("cinecat-downloads")
            .resolve("downloads.json")
            .toFile()
        val registry = DownloadRegistry(file)
        registry.add(movie)
        registry.add(episode)
        registry.recordProgress(movie.itemId, 10.0, 100.0)
        registry.recordProgress(episode.itemId, 20.0, 200.0)
        val sent = mutableListOf<Long>()

        val sync = PendingProgressSync(registry) { pending ->
            if (pending.itemId == movie.itemId) {
                sent += pending.itemId
                true
            } else {
                false
            }
        }

        assertEquals(1, sync.flush())
        assertEquals(listOf(movie.itemId), sent)
        assertEquals(listOf(episode.itemId), registry.pendingProgress().map { it.itemId })
        assertTrue(registry.load().downloads.any { it.itemId == episode.itemId })
    }
}
