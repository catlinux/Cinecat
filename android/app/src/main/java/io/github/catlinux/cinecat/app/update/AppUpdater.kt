package io.github.catlinux.cinecat.app.update

import android.content.Context
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.content.pm.Signature
import android.content.pm.SigningInfo
import android.os.Build
import androidx.annotation.RequiresApi
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CinecatApi
import io.github.catlinux.cinecat.core.api.ClientUpdate
import java.io.File
import java.io.InputStream
import java.security.MessageDigest
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request

const val UPDATE_CHECK_INTERVAL_MILLIS = 12L * 60 * 60 * 1000
const val MAX_UPDATE_BYTES = 200L * 1024 * 1024
const val UPDATE_DOWNLOAD_PREFIX = "https://github.com/catlinux/Cinecat/releases/download/"

enum class UpdateFailure {
    Disabled,
    Network,
    InvalidUrl,
    Download,
    TooLarge,
    HashMismatch,
    InvalidApk,
    PackageMismatch,
    VersionNotNewer,
    SignatureMismatch,
    Install
}

sealed interface UpdateState {
    data object Idle : UpdateState
    data object Checking : UpdateState
    data object UpToDate : UpdateState
    data class Available(val update: ClientUpdate) : UpdateState
    data class Downloading(val progress: Float) : UpdateState
    data class Ready(val update: ClientUpdate, val apkFile: File) : UpdateState
    data class Failed(val reason: UpdateFailure, val detail: String? = null) : UpdateState
    data object Disabled : UpdateState
}

fun isAllowedDownloadUrl(url: String): Boolean {
    return url.startsWith(UPDATE_DOWNLOAD_PREFIX) &&
        !url.contains("..") &&
        !url.contains('@') &&
        !url.contains('?')
}

fun shouldCheck(lastCheckMillis: Long, nowMillis: Long): Boolean {
    return lastCheckMillis <= 0L ||
        nowMillis - lastCheckMillis >= UPDATE_CHECK_INTERVAL_MILLIS
}

fun copyWithSha256(
    input: InputStream,
    target: File,
    expectedSha256: String,
    maxBytes: Long = MAX_UPDATE_BYTES,
    onProgress: (Long) -> Unit = {}
): UpdateFailure? {
    target.parentFile?.mkdirs()
    val digest = MessageDigest.getInstance("SHA-256")
    var total = 0L
    var tooLarge = false
    try {
        target.outputStream().use { output ->
            val buffer = ByteArray(64 * 1024)
            while (true) {
                val read = input.read(buffer)
                if (read < 0) {
                    break
                }
                total += read
                if (total > maxBytes) {
                    tooLarge = true
                    break
                }
                output.write(buffer, 0, read)
                digest.update(buffer, 0, read)
                onProgress(total)
            }
        }
    } catch (_: Exception) {
        target.delete()
        return UpdateFailure.Download
    }
    if (tooLarge) {
        target.delete()
        return UpdateFailure.TooLarge
    }
    val actual = digest.digest().toHex()
    if (!actual.equals(expectedSha256, ignoreCase = true)) {
        target.delete()
        return UpdateFailure.HashMismatch
    }
    return null
}

fun copyAndVerifySha256(
    input: InputStream,
    target: File,
    expectedSha256: String,
    maxBytes: Long = MAX_UPDATE_BYTES,
    onProgress: (Long) -> Unit = {}
): Boolean = copyWithSha256(input, target, expectedSha256, maxBytes, onProgress) == null

class AppUpdater(
    private val context: Context,
    private val apiProvider: () -> CinecatApi?,
    private val downloadClient: OkHttpClient,
    private val preferences: DataStore<Preferences>,
    private val scope: CoroutineScope,
    private val debug: Boolean = false,
    private val now: () -> Long = System::currentTimeMillis
) {
    private val _state = MutableStateFlow<UpdateState>(UpdateState.Idle)
    val state: StateFlow<UpdateState> = _state.asStateFlow()

    private val _autoEnabled = MutableStateFlow(true)
    val autoEnabled: StateFlow<Boolean> = _autoEnabled.asStateFlow()

    private var checkJob: Job? = null
    private var downloadJob: Job? = null

    init {
        if (debug) {
            _state.value = UpdateState.Disabled
        } else {
            scope.launch {
                runCatching {
                    _autoEnabled.value = preferences.data.first()[AUTO_ENABLED] ?: true
                }
            }
        }
    }

    fun check(force: Boolean = false): Job? {
        if (debug) {
            _state.value = UpdateState.Disabled
            return null
        }
        checkJob?.takeIf { it.isActive }?.let { return it }
        val job = scope.launch { performCheck(force) }
        checkJob = job
        return job
    }

    fun checkForUpdates(force: Boolean = false): Job? = check(force)

    fun download(update: ClientUpdate): Job? {
        if (debug) {
            _state.value = UpdateState.Disabled
            return null
        }
        downloadJob?.takeIf { it.isActive }?.let { return it }
        val job = scope.launch { performDownload(update) }
        downloadJob = job
        return job
    }

    fun downloadUpdate(update: ClientUpdate): Job? = download(update)

    suspend fun setAutoEnabled(enabled: Boolean) {
        _autoEnabled.value = enabled
        runCatching {
            preferences.edit { it[AUTO_ENABLED] = enabled }
        }
    }

    // Copiar el APK a la sesión del instalador lleva segundos en la TV: fuera del hilo principal.
    fun install(context: Context, apkFile: File) {
        scope.launch(Dispatchers.IO) {
            try {
                UpdateInstaller.install(context, apkFile)
            } catch (error: Exception) {
                reportInstallFailure(error.message)
            }
        }
    }

    fun reportInstallFailure(detail: String?) {
        _state.value = UpdateState.Failed(UpdateFailure.Install, detail)
    }

    private suspend fun performCheck(force: Boolean) {
        val stored = runCatching { preferences.data.first() }.getOrNull()
        val auto = stored?.get(AUTO_ENABLED) ?: true
        _autoEnabled.value = auto
        if (!force && !auto) {
            updateIdleState(UpdateState.Idle)
            return
        }
        val checkedAt = now()
        val lastCheck = stored?.get(LAST_CHECK) ?: 0L
        if (!force && !shouldCheck(lastCheck, checkedAt)) {
            updateIdleState(UpdateState.UpToDate)
            return
        }

        _state.value = UpdateState.Checking
        val api = apiProvider()
            ?: run {
                _state.value = UpdateState.Failed(UpdateFailure.Network)
                return
            }
        val installedCode = runCatching { installedVersionCode() }.getOrElse {
            _state.value = UpdateState.Failed(UpdateFailure.InvalidApk)
            return
        }
        when (val result = api.clientUpdate(installedCode)) {
            is ApiResult.Success -> {
                saveLastCheck(checkedAt)
                _state.value = result.value?.let(UpdateState::Available) ?: UpdateState.UpToDate
            }
            is ApiResult.HttpError -> {
                _state.value = UpdateState.Failed(UpdateFailure.Network, result.message)
            }
            is ApiResult.NetworkError -> {
                _state.value = UpdateState.Failed(UpdateFailure.Network, result.cause.message)
            }
        }
    }

    private suspend fun performDownload(update: ClientUpdate) {
        if (!isAllowedDownloadUrl(update.url)) {
            _state.value = UpdateState.Failed(UpdateFailure.InvalidUrl)
            return
        }
        val directory = File(context.cacheDir, "updates")
        directory.mkdirs()
        directory.listFiles()
            ?.filter { it.isFile && it.extension.equals("apk", ignoreCase = true) }
            ?.forEach { it.delete() }

        val target = File(directory, "cinecat-${update.version}.apk")
        _state.value = UpdateState.Downloading(0f)
        try {
            val request = Request.Builder().url(update.url).build()
            downloadClient.newCall(request).execute().use { response ->
                if (!response.isSuccessful) {
                    _state.value = UpdateState.Failed(UpdateFailure.Download)
                    return
                }
                val body = response.body
                val total = body.contentLength()
                val failure = copyWithSha256(
                    input = body.byteStream(),
                    target = target,
                    expectedSha256 = update.sha256,
                    maxBytes = MAX_UPDATE_BYTES
                ) { bytes ->
                    if (total > 0) {
                        _state.value = UpdateState.Downloading(
                            (bytes.toFloat() / total).coerceIn(0f, 1f)
                        )
                    }
                }
                if (failure != null) {
                    target.delete()
                    _state.value = UpdateState.Failed(failure)
                    return
                }
            }
        } catch (error: Exception) {
            target.delete()
            _state.value = UpdateState.Failed(UpdateFailure.Download, error.message)
            return
        }

        val verification = verifyApk(target)
        if (verification != null) {
            target.delete()
            _state.value = UpdateState.Failed(verification)
            return
        }
        _state.value = UpdateState.Ready(update, target)
    }

    private fun updateIdleState(value: UpdateState) {
        when (_state.value) {
            UpdateState.Idle, UpdateState.UpToDate, UpdateState.Checking -> {
                _state.value = value
            }
            else -> Unit
        }
    }

    private suspend fun saveLastCheck(millis: Long) {
        runCatching {
            preferences.edit { it[LAST_CHECK] = millis }
        }
    }

    private fun installedVersionCode(): Long {
        val info = context.packageManager.getPackageInfo(context.packageName, 0)
        return info.longVersionCodeCompat()
    }

    private fun verifyApk(apkFile: File): UpdateFailure? {
        return try {
            val manager = context.packageManager
            val flags = signingFlags()
            val archive = manager.getPackageArchiveInfo(apkFile.absolutePath, flags)
                ?: return UpdateFailure.InvalidApk
            if (archive.packageName != context.packageName) {
                return UpdateFailure.PackageMismatch
            }
            val installed = manager.getPackageInfo(context.packageName, flags)
            if (archive.longVersionCodeCompat() <= installed.longVersionCodeCompat()) {
                return UpdateFailure.VersionNotNewer
            }
            if (!sameSignatures(installed, archive)) {
                return UpdateFailure.SignatureMismatch
            }
            null
        } catch (_: Exception) {
            UpdateFailure.InvalidApk
        }
    }

    @Suppress("DEPRECATION")
    private fun sameSignatures(installed: PackageInfo, archive: PackageInfo): Boolean {
        return try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                val installedSigning = installed.signingInfo ?: return false
                val archiveSigning = archive.signingInfo ?: return false
                val installedCerts = signingCertificates(installedSigning)
                val archiveCerts = signingCertificates(archiveSigning)
                installedCerts.isNotEmpty() && installedCerts == archiveCerts
            } else {
                val installedCerts = installed.signatures?.map { it.toByteArray().toHex() }?.toSet()
                val archiveCerts = archive.signatures?.map { it.toByteArray().toHex() }?.toSet()
                !installedCerts.isNullOrEmpty() && installedCerts == archiveCerts
            }
        } catch (_: Exception) {
            false
        }
    }

    @RequiresApi(Build.VERSION_CODES.P)
    private fun signingCertificates(info: SigningInfo): Set<String> {
        val signatures = if (info.hasMultipleSigners()) {
            info.apkContentsSigners
        } else {
            info.signingCertificateHistory
        }
        return signatures.map { it.toByteArray().toHex() }.toSet()
    }

    @Suppress("DEPRECATION")
    private fun signingFlags(): Int {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            PackageManager.GET_SIGNING_CERTIFICATES
        } else {
            PackageManager.GET_SIGNATURES
        }
    }

    @Suppress("DEPRECATION")
    private fun PackageInfo.longVersionCodeCompat(): Long {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            longVersionCode
        } else {
            versionCode.toLong()
        }
    }

    private companion object {
        val LAST_CHECK = longPreferencesKey("app_update_last_check")
        val AUTO_ENABLED = booleanPreferencesKey("app_update_auto")
    }
}

private fun Signature.toHex(): String = toByteArray().toHex()

private fun ByteArray.toHex(): String {
    val hex = "0123456789abcdef"
    return buildString(size * 2) {
        for (byte in this@toHex) {
            val value = byte.toInt() and 0xff
            append(hex[value ushr 4])
            append(hex[value and 0x0f])
        }
    }
}
