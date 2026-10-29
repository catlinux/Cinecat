package io.github.catlinux.cinecat.app

import android.annotation.SuppressLint
import android.content.Context
import android.content.res.Configuration
import android.net.ConnectivityManager
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import coil3.ImageLoader
import coil3.disk.DiskCache
import coil3.disk.directory
import coil3.map.Mapper
import coil3.memory.MemoryCache
import coil3.network.okhttp.OkHttpNetworkFetcherFactory
import io.github.catlinux.cinecat.core.api.LocalAddressDns
import io.github.catlinux.cinecat.core.api.createApi
import io.github.catlinux.cinecat.core.api.createOkHttpClient
import io.github.catlinux.cinecat.core.data.KeystoreTokenCipher
import io.github.catlinux.cinecat.core.data.LibraryRepository
import io.github.catlinux.cinecat.core.data.LocalNetworkMonitor
import io.github.catlinux.cinecat.core.data.PlaybackRepository
import io.github.catlinux.cinecat.core.data.SessionRepository
import io.github.catlinux.cinecat.core.data.SessionStore
import io.github.catlinux.cinecat.core.data.deviceName
import io.github.catlinux.cinecat.core.player.detectCapabilities
import java.io.File
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import okhttp3.Dns
import okhttp3.OkHttpClient

class AppContainer(context: Context) {
    private val appContext = context.applicationContext
    val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val dataStore = PreferenceDataStoreFactory.create(
        scope = appScope,
        produceFile = { File(appContext.filesDir, "cinecat.preferences_pb") }
    )
    private val sessionStore = SessionStore(dataStore, KeystoreTokenCipher())
    private var repositoryRef: SessionRepository? = null

    val localNetworkMonitor: LocalNetworkMonitor
    private val dns: LocalAddressDns
    val client: OkHttpClient

    val sessionRepository: SessionRepository
    val libraryRepository: LibraryRepository
    val playbackRepository: PlaybackRepository
    val imageLoader: ImageLoader

    val caps: String by lazy { calculateCaps(appContext) }

    init {
        val connectivityManager = requireNotNull(
            appContext.getSystemService(ConnectivityManager::class.java)
        )
        localNetworkMonitor = LocalNetworkMonitor(connectivityManager) {
            dns.invalidate()
        }
        dns = LocalAddressDns(
            serverHost = { repositoryRef?.currentServer()?.host },
            localAddress = { repositoryRef?.currentLocalAddress() },
            isLocalNetwork = { localNetworkMonitor.isLocalNetwork() },
            port = { repositoryRef?.currentServer()?.port ?: 443 },
            fallback = Dns.SYSTEM
        )
        val appVersion = BuildConfig.VERSION_NAME
        client = createOkHttpClient(
            serverOrigin = { repositoryRef?.currentServer() },
            token = { repositoryRef?.currentToken() },
            onUnauthorized = { repositoryRef?.onUnauthorized() },
            version = appVersion,
            dns = dns
        )
        val clientName = if (
            isTelevision(
                appContext.resources.configuration.uiMode and Configuration.UI_MODE_TYPE_MASK
            )
        ) {
            "androidtv"
        } else {
            "android"
        }
        sessionRepository = SessionRepository(
            store = sessionStore,
            apiFactory = { origin -> createApi(origin, client) },
            deviceName = deviceName(appContext),
            client = clientName,
            appVersion = appVersion,
            scope = appScope
        )
        repositoryRef = sessionRepository
        libraryRepository = LibraryRepository(
            apiProvider = {
                sessionRepository.currentServer()?.let { origin -> createApi(origin, client) }
            }
        )
        playbackRepository = PlaybackRepository(
            apiProvider = {
                sessionRepository.currentServer()?.let { origin -> createApi(origin, client) }
            }
        )
        imageLoader = ImageLoader.Builder(appContext)
            .components {
                add(OkHttpNetworkFetcherFactory(callFactory = { client }))
                // El servidor devuelve las imágenes con rutas relativas a su origen.
                add(Mapper<String, String> { data, _ ->
                    if (data.startsWith("/")) {
                        sessionRepository.currentServer()?.resolve(data)?.toString()
                    } else {
                        null
                    }
                })
            }
            .memoryCache {
                MemoryCache.Builder()
                    .maxSizePercent(appContext, 0.25)
                    .build()
            }
            .diskCache {
                DiskCache.Builder()
                    .directory(File(appContext.cacheDir, "images"))
                    .maxSizeBytes(100L * 1024 * 1024)
                    .build()
            }
            .build()
        localNetworkMonitor.start()
    }
}

@SuppressLint("UnsafeOptInUsageError")
private fun calculateCaps(context: Context): String = detectCapabilities(context)
