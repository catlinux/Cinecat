package io.github.catlinux.cinecat.app

import android.content.Context
import android.net.Uri
import android.os.Environment
import androidx.core.net.toUri
import androidx.media3.common.MimeTypes
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.offline.Download
import androidx.media3.exoplayer.offline.DownloadManager
import androidx.media3.exoplayer.offline.DownloadRequest
import androidx.media3.exoplayer.offline.DownloadService
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.DownloadRecord
import io.github.catlinux.cinecat.core.api.DownloadRegistry
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.PendingProgressSync
import io.github.catlinux.cinecat.core.api.PlayResponse
import io.github.catlinux.cinecat.core.data.PlaybackRepository
import io.github.catlinux.cinecat.core.data.SessionRepository
import java.io.File
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okhttp3.Request

enum class DownloadPhase {
    Pending,
    Running,
    Completed,
    Failed
}

data class DownloadStatus(
    val phase: DownloadPhase,
    val downloadedBytes: Long = 0,
    val totalBytes: Long = 0,
    val percentValue: Int? = null,
    val waitingForServer: Boolean = false
) {
    val percent: Int?
        get() = percentValue ?: downloadPercent(downloadedBytes, totalBytes)
}

sealed interface DownloadStartResult {
    data class Started(val record: DownloadRecord) : DownloadStartResult
    data class Failed(val message: String? = null) : DownloadStartResult
}

private const val STATUS_POLL_MS = 1_000L

@UnstableApi
class DownloadRepository(
    private val context: Context,
    private val registry: DownloadRegistry,
    private val playbackRepository: PlaybackRepository,
    private val sessionRepository: SessionRepository,
    private val client: OkHttpClient,
    offline: OfflineDownloads,
    private val scope: CoroutineScope,
    private val caps: () -> String
) {
    private val manager: DownloadManager = offline.downloadManager
    val records = registry.state

    private val _completed = MutableStateFlow<Set<Long>>(emptySet())
    val completed: StateFlow<Set<Long>> = _completed.asStateFlow()

    // Una sola re-firma por descarga y sesión: si el fallo no era la firma, no se entra en bucle.
    private val resigned = mutableSetOf<Long>()

    // Cambia con cada evento de Media3 para despertar el sondeo cuando no hay nada activo.
    private val changes = MutableStateFlow(0)

    /** Estado de cada descarga por id de ficha; solo sondea mientras alguien lo observa. */
    val statuses: StateFlow<Map<Long, DownloadStatus>> =
        pollStatuses().stateIn(scope, SharingStarted.WhileSubscribed(5_000), emptyMap())

    init {
        manager.addListener(object : DownloadManager.Listener {
            override fun onDownloadChanged(
                downloadManager: DownloadManager,
                download: Download,
                finalException: Exception?
            ) {
                changes.value++
                val itemId = download.request.id.toLongOrNull() ?: return
                updateCompleted(itemId, download.state == Download.STATE_COMPLETED)
                if (download.state == Download.STATE_COMPLETED) {
                    scope.launch { registry.updateSize(itemId, download.bytesDownloaded) }
                } else if (download.state == Download.STATE_FAILED && resigned.add(itemId)) {
                    scope.launch { resign(itemId) }
                }
            }

            override fun onDownloadRemoved(downloadManager: DownloadManager, download: Download) {
                changes.value++
                download.request.id.toLongOrNull()?.let { updateCompleted(it, false) }
            }
        })
    }

    suspend fun initialize() {
        registry.load()
        val downloads = withContext(Dispatchers.IO) { currentDownloads() }
        _completed.value = downloads
            .filter { it.state == Download.STATE_COMPLETED }
            .mapNotNull { it.request.id.toLongOrNull() }
            .toSet()
        // La firma pudo caducar mientras la app estaba cerrada.
        downloads
            .filter { it.state != Download.STATE_COMPLETED }
            .mapNotNull { it.request.id.toLongOrNull() }
            .forEach {
                resigned.add(it)
                resign(it)
            }
        flushPending()
        migrateLegacyDownloads()
    }

    suspend fun start(item: MediaItem, caps: String): DownloadStartResult {
        return when (val playback = playbackRepository.play(item.id, caps = caps, download = true)) {
            is ApiResult.Success -> enqueue(item, playback.value)
            is ApiResult.HttpError -> DownloadStartResult.Failed(playback.message)
            is ApiResult.NetworkError -> DownloadStartResult.Failed(playback.cause.message)
        }
    }

    /** Como [start], pero en el ámbito de la app: sobrevive a que la pantalla se recargue (p. ej. tras el diálogo de permisos). */
    fun startAsync(item: MediaItem, caps: String, onResult: (DownloadStartResult) -> Unit) {
        scope.launch { onResult(start(item, caps)) }
    }

    private fun pollStatuses(): Flow<Map<Long, DownloadStatus>> = flow {
        val lastBytes = mutableMapOf<Long, Long>()
        val unchanged = mutableMapOf<Long, Int>()
        while (true) {
            val seen = changes.value
            val downloads = withContext(Dispatchers.IO) { currentDownloads() }
            val map = downloads.mapNotNull { download ->
                val itemId = download.request.id.toLongOrNull() ?: return@mapNotNull null
                val bytes = download.bytesDownloaded
                unchanged[itemId] = if (lastBytes[itemId] == bytes) (unchanged[itemId] ?: 0) + 1 else 0
                lastBytes[itemId] = bytes
                val phase = downloadPhase(download.state)
                itemId to DownloadStatus(
                    phase = phase,
                    downloadedBytes = bytes,
                    percentValue = download.percentDownloaded.takeIf { it >= 0f }?.toInt(),
                    waitingForServer = isWaitingForServer(phase, unchanged[itemId] ?: 0)
                )
            }.toMap()
            lastBytes.keys.retainAll(map.keys)
            unchanged.keys.retainAll(map.keys)
            emit(map)
            if (map.values.any { it.phase == DownloadPhase.Pending || it.phase == DownloadPhase.Running }) {
                delay(STATUS_POLL_MS)
            } else {
                changes.first { it != seen }
            }
        }
    }

    suspend fun cancel(record: DownloadRecord) = delete(record)

    suspend fun delete(record: DownloadRecord) {
        DownloadService.sendRemoveDownload(
            context,
            CinecatDownloadService::class.java,
            record.itemId.toString(),
            false
        )
        withContext(Dispatchers.IO) { subtitlesDir(record.itemId).deleteRecursively() }
        updateCompleted(record.itemId, false)
        registry.remove(record.itemId)
    }

    suspend fun saveLocalProgress(itemId: Long, positionSeconds: Double, durationSeconds: Double) {
        registry.recordProgress(itemId, positionSeconds, durationSeconds)
        if (playbackRepository.saveProgress(itemId, positionSeconds, durationSeconds) is ApiResult.Success) {
            registry.clearPending(itemId)
        }
    }

    suspend fun flushPending() {
        PendingProgressSync(registry) { pending ->
            playbackRepository.saveProgress(
                pending.itemId,
                pending.positionSeconds,
                pending.durationSeconds
            ) is ApiResult.Success
        }.flush()
    }

    fun subtitlesDir(itemId: Long): File = File(context.filesDir, "offline-subs/$itemId")

    fun hasLocalFile(record: DownloadRecord): Boolean = record.itemId in _completed.value

    fun localUri(record: DownloadRecord): String =
        manager.downloadIndex.getDownload(record.itemId.toString())?.request?.uri?.toString().orEmpty()

    /** Subtítulos de texto guardados con la descarga, como fuentes locales del reproductor. */
    fun localSubtitles(itemId: Long): List<PlayerSubtitle> {
        val data = manager.downloadIndex.getDownload(itemId.toString())?.request?.data
        val dir = subtitlesDir(itemId)
        return decodeOfflineMeta(data)?.toPlayerSubtitles { Uri.fromFile(File(dir, it)).toString() }
            .orEmpty()
    }

    // Las descargas anteriores a la 0.67.0 eran el archivo original con el DownloadManager del
    // sistema: se cancelan, se borra el archivo y se quita el registro.
    private suspend fun migrateLegacyDownloads() {
        val legacy = registry.state.value.downloads.filter { it.fileName.isNotBlank() }
        if (legacy.isEmpty()) {
            return
        }
        withContext(Dispatchers.IO) {
            val system = context.getSystemService(android.app.DownloadManager::class.java)
            val movies = context.getExternalFilesDir(Environment.DIRECTORY_MOVIES) ?: context.filesDir
            legacy.forEach { record ->
                if (record.downloadId != 0L) {
                    runCatching { system?.remove(record.downloadId) }
                }
                File(movies, record.fileName).delete()
                File(context.filesDir, record.fileName).delete()
            }
        }
        legacy.forEach { registry.remove(it.itemId) }
    }

    private fun currentDownloads(): List<Download> =
        manager.downloadIndex.getDownloads().use { cursor ->
            buildList { while (cursor.moveToNext()) add(cursor.download) }
        }

    private fun updateCompleted(itemId: Long, done: Boolean) {
        _completed.value = if (done) _completed.value + itemId else _completed.value - itemId
    }

    private suspend fun enqueue(item: MediaItem, playback: PlayResponse): DownloadStartResult {
        val origin = sessionRepository.currentServer() ?: return DownloadStartResult.Failed()
        val playlist = origin.resolve(playback.url)?.toString()
            ?: return DownloadStartResult.Failed()
        val title = downloadDisplayTitle(item).ifBlank { context.getString(R.string.app_name) }
        val poster = item.posterUrl ?: item.seriesPosterUrl
        val meta = OfflineMeta(
            itemId = item.id,
            title = title,
            seriesTitle = item.seriesTitle,
            season = item.season,
            episode = item.episode,
            episodeTitle = item.episodeTitle,
            posterUrl = poster,
            subtitles = fetchSubtitles(item.id, playback)
        )
        send(buildRequest(item.id, playlist, meta))
        val record = DownloadRecord(
            itemId = item.id,
            kind = item.kind,
            title = title,
            seriesTitle = item.seriesTitle,
            season = item.season,
            episode = item.episode,
            episodeTitle = item.episodeTitle,
            posterUrl = poster
        )
        registry.add(record)
        resigned.remove(item.id)
        return DownloadStartResult.Started(record)
    }

    // Vuelve a pedir la URL firmada y la añade con el mismo id: Media3 fusiona la
    // petición, adopta la URI nueva y reutiliza los segmentos ya descargados.
    private suspend fun resign(itemId: Long) {
        val existing = withContext(Dispatchers.IO) {
            manager.downloadIndex.getDownload(itemId.toString())
        } ?: return
        if (existing.state == Download.STATE_COMPLETED) {
            return
        }
        val playback = (
            playbackRepository.play(itemId, caps = caps(), download = true) as? ApiResult.Success
            )?.value ?: return
        val origin = sessionRepository.currentServer() ?: return
        val uri = origin.resolve(playback.url)?.toString() ?: return
        val old = existing.request
        send(
            DownloadRequest.Builder(old.id, uri.toUri())
                .setMimeType(old.mimeType)
                .setStreamKeys(old.streamKeys)
                .setCustomCacheKey(old.customCacheKey)
                .setData(old.data)
                .build()
        )
    }

    private fun send(request: DownloadRequest) {
        runCatching {
            DownloadService.sendAddDownload(
                context,
                CinecatDownloadService::class.java,
                request,
                true
            )
        }.onFailure { manager.addDownload(request) }
    }

    private fun buildRequest(itemId: Long, playlist: String, meta: OfflineMeta): DownloadRequest =
        DownloadRequest.Builder(itemId.toString(), playlist.toUri())
            .setMimeType(MimeTypes.APPLICATION_M3U8)
            .setData(encodeOfflineMeta(meta))
            .build()

    private suspend fun fetchSubtitles(
        itemId: Long,
        playback: PlayResponse
    ): List<OfflineSubtitle> = withContext(Dispatchers.IO) {
        val origin = sessionRepository.currentServer() ?: return@withContext emptyList()
        val dir = subtitlesDir(itemId)
        dir.mkdirs()
        playback.subtitles.filter { !it.burnIn && it.url.isNotBlank() }.mapNotNull { option ->
            val url = origin.resolve(option.url) ?: return@mapNotNull null
            val fileName = "${option.id.filter { it.isLetterOrDigit() }.ifEmpty { "s" }}.vtt"
            val saved = runCatching {
                client.newCall(Request.Builder().url(url).build()).execute().use { response ->
                    if (!response.isSuccessful) {
                        return@use false
                    }
                    File(dir, fileName).outputStream().use { out ->
                        response.body.byteStream().copyTo(out)
                    }
                    true
                }
            }.getOrDefault(false)
            if (!saved) {
                return@mapNotNull null
            }
            OfflineSubtitle(
                id = option.id,
                label = option.title.ifBlank { option.languageName }.ifBlank { option.language },
                language = option.language,
                file = fileName
            )
        }
    }
}
