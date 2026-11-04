package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.MediaItem
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DownloadFormattingTest {
    @Test
    fun componeElTituloDePeliculaYEpisodio() {
        val movie = MediaItem(id = 1, title = "Película", kind = "movie")
        assertEquals("Película", downloadDisplayTitle(movie))

        val episode = MediaItem(
            id = 2,
            title = "Episodio",
            kind = "episode",
            seriesTitle = "Serie",
            season = 2,
            episode = 3,
            episodeTitle = "El capítulo"
        )
        assertEquals("Serie · T2E3 · El capítulo", downloadDisplayTitle(episode))
    }

    @Test
    fun calculaElPorcentajeYElTamano() {
        assertEquals(25, downloadPercent(25, 100))
        assertEquals(100, downloadPercent(150, 100))
        assertNull(downloadPercent(10, 0))
        assertEquals("2.0 KB", formatDownloadSize(2048))
        assertEquals("", formatDownloadSize(0))
    }

    @Test
    fun etiquetaLosEstadosDeLaDescarga() {
        assertEquals(DownloadLabel.Download, downloadLabel(false, false, null))
        assertEquals(DownloadLabel.Preparing, downloadLabel(false, true, null))
        assertEquals(DownloadLabel.Downloading, downloadLabel(true, false, DownloadPhase.Running))
        assertEquals(DownloadLabel.Waiting, downloadLabel(true, false, DownloadPhase.Running, true))
        assertEquals(DownloadLabel.Downloaded, downloadLabel(true, false, DownloadPhase.Completed, true))
        assertEquals(DownloadLabel.Failed, downloadLabel(true, false, DownloadPhase.Failed))
    }
}
