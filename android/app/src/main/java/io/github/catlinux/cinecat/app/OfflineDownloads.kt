package io.github.catlinux.cinecat.app

import android.content.Context
import androidx.media3.common.util.UnstableApi
import androidx.media3.database.StandaloneDatabaseProvider
import androidx.media3.datasource.DataSpec
import androidx.media3.datasource.DefaultDataSource
import androidx.media3.datasource.cache.Cache
import androidx.media3.datasource.cache.CacheDataSource
import androidx.media3.datasource.cache.CacheKeyFactory
import androidx.media3.datasource.cache.NoOpCacheEvictor
import androidx.media3.datasource.cache.SimpleCache
import androidx.media3.datasource.okhttp.OkHttpDataSource
import androidx.media3.exoplayer.offline.DefaultDownloadIndex
import androidx.media3.exoplayer.offline.DefaultDownloaderFactory
import androidx.media3.exoplayer.offline.Download
import androidx.media3.exoplayer.offline.DownloadManager
import io.github.catlinux.cinecat.core.api.CinecatJson
import java.io.File
import java.util.concurrent.Executors
import kotlinx.serialization.Serializable
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient

@Serializable
data class OfflineSubtitle(
    val id: String,
    val label: String,
    val language: String = "",
    val file: String
)

/** Metadatos que viajan en la petición de Media3 (`DownloadRequest.data`). */
@Serializable
data class OfflineMeta(
    val itemId: Long,
    val title: String = "",
    val seriesTitle: String? = null,
    val season: Int? = null,
    val episode: Int? = null,
    val episodeTitle: String? = null,
    val posterUrl: String? = null,
    val subtitles: List<OfflineSubtitle> = emptyList()
)

fun encodeOfflineMeta(meta: OfflineMeta): ByteArray =
    CinecatJson.encodeToString(OfflineMeta.serializer(), meta).toByteArray(Charsets.UTF_8)

fun decodeOfflineMeta(data: ByteArray?): OfflineMeta? {
    if (data == null || data.isEmpty()) {
        return null
    }
    return runCatching {
        CinecatJson.decodeFromString(OfflineMeta.serializer(), String(data, Charsets.UTF_8))
    }.getOrNull()
}

/**
 * Clave de caché sin la firma: una URL firmada de nuevo (`exp`/`sig`) reutiliza
 * lo ya descargado. El resto de parámetros (audio, calidad…) sí identifican el
 * contenido y se ordenan para que no dependan del orden de la URL.
 */
fun offlineCacheKey(url: String): String {
    val parsed = url.toHttpUrlOrNull() ?: return url
    val names = parsed.queryParameterNames
        .filter { it != "exp" && it != "sig" }
        .sorted()
    val builder = parsed.newBuilder().query(null)
    for (name in names) {
        for (value in parsed.queryParameterValues(name)) {
            builder.addQueryParameter(name, value)
        }
    }
    return builder.build().toString()
}

fun OfflineMeta.toPlayerSubtitles(fileUrl: (String) -> String): List<PlayerSubtitle> =
    subtitles.map { subtitle ->
        PlayerSubtitle(
            url = fileUrl(subtitle.file),
            language = subtitle.language,
            label = subtitle.label,
            selected = false,
            id = subtitle.id
        )
    }

private const val WAITING_POLLS = 5

/** Media3 reintenta solo cuando el servidor responde 503: sin avance durante unos sondeos es espera. */
fun isWaitingForServer(phase: DownloadPhase, unchangedPolls: Int): Boolean =
    phase == DownloadPhase.Running && unchangedPolls >= WAITING_POLLS

fun downloadPhase(state: Int): DownloadPhase = when (state) {
    Download.STATE_COMPLETED -> DownloadPhase.Completed
    Download.STATE_FAILED -> DownloadPhase.Failed
    Download.STATE_DOWNLOADING -> DownloadPhase.Running
    else -> DownloadPhase.Pending
}

@UnstableApi
class OfflineDownloads(private val context: Context, client: OkHttpClient) {
    private val databaseProvider = StandaloneDatabaseProvider(context)
    val cache: Cache = SimpleCache(
        File(context.filesDir, "offline-hls"),
        NoOpCacheEvictor(),
        databaseProvider
    )
    val upstreamFactory = OkHttpDataSource.Factory(client)
    val cacheKeyFactory = CacheKeyFactory { spec: DataSpec ->
        spec.key ?: offlineCacheKey(spec.uri.toString())
    }

    /** Fuente de datos para reproducir lo descargado; la red solo se usa si falta algo. */
    val playbackDataSourceFactory: CacheDataSource.Factory = CacheDataSource.Factory()
        .setCache(cache)
        .setCacheKeyFactory(cacheKeyFactory)
        .setUpstreamDataSourceFactory(DefaultDataSource.Factory(context, upstreamFactory))
        .setCacheWriteDataSinkFactory(null)
        .setFlags(CacheDataSource.FLAG_IGNORE_CACHE_ON_ERROR)

    val downloadManager: DownloadManager = DownloadManager(
        context,
        DefaultDownloadIndex(databaseProvider),
        DefaultDownloaderFactory(
            CacheDataSource.Factory()
                .setCache(cache)
                .setCacheKeyFactory(cacheKeyFactory)
                .setUpstreamDataSourceFactory(upstreamFactory),
            Executors.newFixedThreadPool(2)
        )
    ).apply { maxParallelDownloads = 1 }
}
