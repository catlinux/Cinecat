package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.PlayResponse
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
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
    fun obtieneLaExtensionDelNombreOdelTipoDeContenido() {
        assertEquals(
            "mkv",
            downloadExtension(
                "attachment; filename*=UTF-8''Pel%C3%ADcula%20(2020).MKV",
                "application/octet-stream"
            )
        )
        assertEquals("mp4", downloadExtension(null, "video/mp4"))
        assertEquals("bin", downloadExtension("attachment; filename=sin-extension", null))
        assertEquals("mkv", downloadExtension("attachment; filename=otra.mkv", null))
        assertEquals("7.mkv", downloadFileName(7, null, "video/x-matroska"))
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
    fun soloSeDescargaLaReproduccionDirecta() {
        assertTrue(canDownload(PlayResponse(url = "/stream", mode = "direct")))
        assertFalse(canDownload(PlayResponse(url = "/stream", mode = "remux")))
    }

    @Test
    fun etiquetaLosEstadosDeLaDescarga() {
        assertEquals(DownloadLabel.Download, downloadLabel(false, false, null))
        assertEquals(DownloadLabel.Preparing, downloadLabel(false, true, null))
        assertEquals(DownloadLabel.Downloading, downloadLabel(true, false, DownloadPhase.Running))
        assertEquals(DownloadLabel.Downloaded, downloadLabel(true, false, DownloadPhase.Completed))
        assertEquals(DownloadLabel.Failed, downloadLabel(true, false, DownloadPhase.Failed))
    }
}
