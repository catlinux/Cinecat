package io.github.catlinux.cinecat.app

import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.offline.Download
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

@UnstableApi
class OfflineDownloadsTest {
    @Test
    fun laClaveDeCacheIgnoraLaFirma() {
        val first = "https://cine.example/api/v1/stream/7/seg0.ts?exp=1&sig=aaa&caps=h264&audio=1&q=720"
        val second = "https://cine.example/api/v1/stream/7/seg0.ts?q=720&audio=1&sig=bbb&caps=h264&exp=2"

        assertEquals(offlineCacheKey(first), offlineCacheKey(second))
        assertEquals(
            "https://cine.example/api/v1/stream/7/seg0.ts?audio=1&caps=h264&q=720",
            offlineCacheKey(first)
        )
    }

    @Test
    fun laClaveDistingueAudioYCalidad() {
        val base = "https://cine.example/api/v1/stream/7/playlist.m3u8?exp=1&sig=a&audio=0"
        val other = "https://cine.example/api/v1/stream/7/playlist.m3u8?exp=1&sig=a&audio=1"

        assert(offlineCacheKey(base) != offlineCacheKey(other))
    }

    @Test
    fun laClaveSinConsultaSeConserva() {
        assertEquals(
            "https://cine.example/a/b.ts",
            offlineCacheKey("https://cine.example/a/b.ts?exp=1&sig=x")
        )
    }

    @Test
    fun mapeaLosEstadosDeMedia3() {
        assertEquals(DownloadPhase.Completed, downloadPhase(Download.STATE_COMPLETED))
        assertEquals(DownloadPhase.Failed, downloadPhase(Download.STATE_FAILED))
        assertEquals(DownloadPhase.Running, downloadPhase(Download.STATE_DOWNLOADING))
        assertEquals(DownloadPhase.Pending, downloadPhase(Download.STATE_QUEUED))
        assertEquals(DownloadPhase.Pending, downloadPhase(Download.STATE_STOPPED))
        assertEquals(DownloadPhase.Pending, downloadPhase(Download.STATE_RESTARTING))
    }

    @Test
    fun losMetadatosDeLaPeticionSeRecuperan() {
        val meta = OfflineMeta(
            itemId = 9,
            title = "Serie · T1E2 · Piloto",
            seriesTitle = "Serie",
            season = 1,
            episode = 2,
            subtitles = listOf(OfflineSubtitle("e0", "Español", "es", "e0.vtt"))
        )

        assertEquals(meta, decodeOfflineMeta(encodeOfflineMeta(meta)))
        assertNull(decodeOfflineMeta(null))
        assertNull(decodeOfflineMeta("no es json".toByteArray()))
    }

    @Test
    fun elPorcentajeDeMedia3TienePrioridad() {
        assertEquals(42, DownloadStatus(DownloadPhase.Running, percentValue = 42).percent)
        assertEquals(50, DownloadStatus(DownloadPhase.Running, 5, 10).percent)
        assertNull(DownloadStatus(DownloadPhase.Running).percent)
    }
}
