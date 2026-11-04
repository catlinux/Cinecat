package io.github.catlinux.cinecat.app

import android.app.DownloadManager
import android.content.Context
import android.net.Uri
import android.os.Environment
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.DownloadRecord
import io.github.catlinux.cinecat.core.api.DownloadRegistry
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.PendingProgressSync
import io.github.catlinux.cinecat.core.data.PlaybackRepository
import io.github.catlinux.cinecat.core.data.SessionRepository
import java.io.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl
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
    val totalBytes: Long = 0
) {
    val percent: Int?
        get() = downloadPercent(downloadedBytes, totalBytes)
}

sealed interface DownloadStartResult {
    data class Started(val record: DownloadRecord) : DownloadStartResult
    data class NotDownloadable(val reason: String?) : DownloadStartResult
    data class Failed(val message: String? = null) : DownloadStartResult
}

private data class RemoteFileInfo(
    val extension: String,
    val sizeBytes: Long
)

class DownloadRepository(
    private val context: Context,
    private val registry: DownloadRegistry,
    private val playbackRepository: PlaybackRepository,
    private val sessionRepository: SessionRepository,
    private val client: OkHttpClient
) {
    private val manager = requireNotNull(context.getSystemService(DownloadManager::class.java))
    val records = registry.state

    suspend fun initialize() {
        registry.load()
        flushPending()
    }

    suspend fun start(item: MediaItem, caps: String): DownloadStartResult {
        return when (val playback = playbackRepository.play(item.id, caps = caps)) {
            is ApiResult.Success -> {
                if (canDownload(playback.value)) {
                    enqueue(item)
                } else {
                    DownloadStartResult.NotDownloadable(playback.value.downloadBlocked)
                }
            }
            is ApiResult.HttpError -> DownloadStartResult.Failed(playback.message)
            is ApiResult.NetworkError -> DownloadStartResult.Failed(playback.cause.message)
        }
    }

    suspend fun status(record: DownloadRecord): DownloadStatus = withContext(Dispatchers.IO) {
        val query = DownloadManager.Query().setFilterById(record.downloadId)
        manager.query(query).use { cursor ->
            if (!cursor.moveToFirst()) {
                val file = localFile(record)
                if (file.isFile && (record.sizeBytes <= 0L || file.length() >= record.sizeBytes)) {
                    return@withContext DownloadStatus(
                        DownloadPhase.Completed,
                        file.length(),
                        file.length()
                    )
                }
                return@withContext DownloadStatus(DownloadPhase.Failed)
            }
            val status = cursor.getInt(cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_STATUS))
            val downloaded = cursor.getLong(
                cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_BYTES_DOWNLOADED_SO_FAR)
            )
            val total = cursor.getLong(
                cursor.getColumnIndexOrThrow(DownloadManager.COLUMN_TOTAL_SIZE_BYTES)
            ).takeIf { it > 0L } ?: record.sizeBytes
            val phase = when (status) {
                DownloadManager.STATUS_SUCCESSFUL -> {
                    if (localFile(record).isFile) {
                        if (total > 0L && record.sizeBytes != total) {
                            registry.updateSize(record.itemId, total)
                        }
                        DownloadPhase.Completed
                    } else {
                        DownloadPhase.Failed
                    }
                }
                DownloadManager.STATUS_FAILED -> DownloadPhase.Failed
                DownloadManager.STATUS_PAUSED -> DownloadPhase.Pending
                else -> DownloadPhase.Running
            }
            DownloadStatus(phase, downloaded, total)
        }
    }

    suspend fun cancel(record: DownloadRecord) {
        manager.remove(record.downloadId)
        localFile(record).delete()
        registry.remove(record.itemId)
    }

    suspend fun delete(record: DownloadRecord) {
        manager.remove(record.downloadId)
        localFile(record).delete()
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

    fun localFile(record: DownloadRecord): File {
        val movies = context.getExternalFilesDir(Environment.DIRECTORY_MOVIES)
        return File(movies ?: context.filesDir, record.fileName)
    }

    fun localUri(record: DownloadRecord): String = Uri.fromFile(localFile(record)).toString()

    fun hasLocalFile(record: DownloadRecord): Boolean = localFile(record).isFile

    private suspend fun enqueue(item: MediaItem): DownloadStartResult {
        val origin = sessionRepository.currentServer()
            ?: return DownloadStartResult.Failed()
        val token = sessionRepository.currentToken()
            ?: return DownloadStartResult.Failed()
        val downloadUrl = origin.newBuilder()
            .addPathSegments("api/v1/items/${item.id}/download")
            .build()
        val remote = remoteFileInfo(downloadUrl, token)
        val fileName = "${item.id}.${remote.extension}"
        val title = downloadDisplayTitle(item).ifBlank {
            context.getString(R.string.app_name)
        }
        val request = DownloadManager.Request(Uri.parse(downloadUrl.toString()))
            .setTitle(title)
            .setNotificationVisibility(
                DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED
            )
            .setDestinationInExternalFilesDir(
                context,
                Environment.DIRECTORY_MOVIES,
                fileName
            )
            .addRequestHeader("Authorization", "Bearer $token")
            .setAllowedOverMetered(true)
            .setAllowedOverRoaming(false)
        val downloadId = runCatching { manager.enqueue(request) }.getOrNull()
            ?: return DownloadStartResult.Failed()
        val record = DownloadRecord(
            itemId = item.id,
            kind = item.kind,
            title = title,
            seriesTitle = item.seriesTitle,
            season = item.season,
            episode = item.episode,
            episodeTitle = item.episodeTitle,
            downloadId = downloadId,
            fileName = fileName,
            sizeBytes = remote.sizeBytes,
            posterUrl = item.posterUrl ?: item.seriesPosterUrl
        )
        registry.add(record)
        return DownloadStartResult.Started(record)
    }

    private suspend fun remoteFileInfo(url: HttpUrl, token: String): RemoteFileInfo =
        withContext(Dispatchers.IO) {
            val request = Request.Builder()
                .url(url)
                .head()
                .header("Authorization", "Bearer $token")
                .build()
            val response = runCatching { client.newCall(request).execute() }.getOrNull()
                ?: return@withContext RemoteFileInfo("bin", 0)
            response.use {
                if (!it.isSuccessful) {
                    return@withContext RemoteFileInfo("bin", 0)
                }
                RemoteFileInfo(
                    extension = downloadExtension(
                        it.header("Content-Disposition"),
                        it.header("Content-Type")
                    ),
                    sizeBytes = it.header("Content-Length")?.toLongOrNull() ?: 0
                )
            }
        }
}
