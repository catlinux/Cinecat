package io.github.catlinux.cinecat.core.api

import java.io.File
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

@Serializable
data class DownloadRecord(
    val itemId: Long,
    val kind: String = "",
    val title: String = "",
    val seriesTitle: String? = null,
    val season: Int? = null,
    val episode: Int? = null,
    val episodeTitle: String? = null,
    val downloadId: Long = 0,
    val fileName: String = "",
    val sizeBytes: Long = 0,
    val posterUrl: String? = null,
    val positionSeconds: Double = 0.0,
    val durationSeconds: Double = 0.0
)

@Serializable
data class PendingProgress(
    val itemId: Long,
    val positionSeconds: Double,
    val durationSeconds: Double
)

@Serializable
data class DownloadsState(
    val downloads: List<DownloadRecord> = emptyList(),
    val pendingProgress: List<PendingProgress> = emptyList()
)

/**
 * Registro local compartido por la ficha, la pantalla de descargas y el
 * reproductor. Se guarda como JSON en la carpeta privada de la app.
 */
class DownloadRegistry(
    private val file: File,
    private val json: Json = CinecatJson,
    private val io: CoroutineDispatcher = Dispatchers.IO
) {
    private val mutex = Mutex()
    private val _state = MutableStateFlow(DownloadsState())
    val state: StateFlow<DownloadsState> = _state.asStateFlow()

    @Volatile
    private var loaded = false

    suspend fun load(): DownloadsState = mutex.withLock {
        ensureLoaded()
    }

    suspend fun add(record: DownloadRecord): DownloadsState = mutate { current ->
        val downloads = current.downloads.toMutableList()
        val index = downloads.indexOfFirst { it.itemId == record.itemId }
        if (index >= 0) {
            downloads[index] = record
        } else {
            downloads += record
        }
        current.copy(downloads = downloads)
    }

    suspend fun remove(itemId: Long): DownloadsState = mutate { current ->
        current.copy(
            downloads = current.downloads.filterNot { it.itemId == itemId },
            pendingProgress = current.pendingProgress.filterNot { it.itemId == itemId }
        )
    }

    suspend fun updateSize(itemId: Long, sizeBytes: Long): DownloadsState = mutate { current ->
        current.copy(
            downloads = current.downloads.map { record ->
                if (record.itemId == itemId) record.copy(sizeBytes = sizeBytes) else record
            }
        )
    }

    suspend fun recordProgress(
        itemId: Long,
        positionSeconds: Double,
        durationSeconds: Double
    ): DownloadsState = mutate { current ->
        current.copy(
            downloads = current.downloads.map { record ->
                if (record.itemId == itemId) {
                    record.copy(
                        positionSeconds = positionSeconds,
                        durationSeconds = durationSeconds
                    )
                } else {
                    record
                }
            },
            pendingProgress = current.pendingProgress
                .filterNot { it.itemId == itemId } +
                PendingProgress(itemId, positionSeconds, durationSeconds)
        )
    }

    suspend fun pendingProgress(): List<PendingProgress> = load().pendingProgress

    suspend fun clearPending(itemId: Long): DownloadsState = mutate { current ->
        current.copy(
            pendingProgress = current.pendingProgress.filterNot { it.itemId == itemId }
        )
    }

    private suspend fun mutate(block: (DownloadsState) -> DownloadsState): DownloadsState =
        mutex.withLock {
            val current = ensureLoaded()
            val next = block(current)
            withContext(io) { write(next) }
            _state.value = next
            next
        }

    private suspend fun ensureLoaded(): DownloadsState {
        if (loaded) {
            return _state.value
        }
        val current = withContext(io) { read() }
        _state.value = current
        loaded = true
        return current
    }

    private fun read(): DownloadsState {
        if (!file.isFile) {
            return DownloadsState()
        }
        return runCatching {
            json.decodeFromString(DownloadsState.serializer(), file.readText())
        }.getOrDefault(DownloadsState())
    }

    private fun write(value: DownloadsState) {
        val text = json.encodeToString(DownloadsState.serializer(), value)
        val parent = file.parentFile
        parent?.mkdirs()
        val temporary = File(parent, file.name + ".tmp")
        temporary.writeText(text)
        if (!temporary.renameTo(file)) {
            // Algunos sistemas no sobrescriben al renombrar; el archivo pequeño
            // se escribe directamente para no perder la última posición.
            file.writeText(text)
            temporary.delete()
        }
    }
}

class PendingProgressSync(
    private val registry: DownloadRegistry,
    private val send: suspend (PendingProgress) -> Boolean
) {
    suspend fun flush(): Int {
        var sent = 0
        for (pending in registry.pendingProgress()) {
            val success = runCatching { send(pending) }.getOrDefault(false)
            if (success) {
                registry.clearPending(pending.itemId)
                sent++
            }
        }
        return sent
    }
}
