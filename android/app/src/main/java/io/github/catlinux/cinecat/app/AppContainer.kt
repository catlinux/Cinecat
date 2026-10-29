package io.github.catlinux.cinecat.app

import android.content.Context
import android.content.res.Configuration
import android.net.ConnectivityManager
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import io.github.catlinux.cinecat.core.api.LocalAddressDns
import io.github.catlinux.cinecat.core.api.createApi
import io.github.catlinux.cinecat.core.api.createOkHttpClient
import io.github.catlinux.cinecat.core.data.KeystoreTokenCipher
import io.github.catlinux.cinecat.core.data.LocalNetworkMonitor
import io.github.catlinux.cinecat.core.data.SessionRepository
import io.github.catlinux.cinecat.core.data.SessionStore
import io.github.catlinux.cinecat.core.data.deviceName
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
    private val client: OkHttpClient

    val sessionRepository: SessionRepository

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
        localNetworkMonitor.start()
    }
}
