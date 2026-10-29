package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ApiResult
import io.github.catlinux.cinecat.core.api.CinecatApi
import io.github.catlinux.cinecat.core.api.LoginRequest
import io.github.catlinux.cinecat.core.api.PairCreateRequest
import io.github.catlinux.cinecat.core.api.PairTokenRequest
import java.net.InetAddress
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.launch
import okhttp3.HttpUrl

sealed interface SessionState {
    data object Loading : SessionState
    data object NoServer : SessionState
    data class SignedOut(val server: HttpUrl) : SessionState
    data class SignedIn(val server: HttpUrl, val user: String, val role: String) : SessionState
}

sealed interface LoginResult {
    data object Success : LoginResult
    data object InvalidCredentials : LoginResult
    data class RateLimited(val retryAfterSeconds: Long?) : LoginResult
    data class Failure(val message: String?) : LoginResult
}

sealed interface PairingState {
    data class Waiting(
        val code: String,
        val verificationUrl: String,
        val expiresAtMillis: Long
    ) : PairingState

    data class Approved(val user: String, val role: String) : PairingState
    data object Expired : PairingState
    data class Failed(val message: String?) : PairingState
}

class SessionRepository(
    private val store: SessionStore,
    private val apiFactory: (HttpUrl) -> CinecatApi,
    private val deviceName: String,
    private val client: String,
    private val appVersion: String,
    private val scope: CoroutineScope,
    private val clock: () -> Long = System::currentTimeMillis
) {
    private val _state = MutableStateFlow<SessionState>(SessionState.Loading)
    val state: StateFlow<SessionState> = _state.asStateFlow()

    @Volatile
    private var server: HttpUrl? = null

    @Volatile
    private var localAddress: InetAddress? = null

    @Volatile
    private var token: String? = null

    init {
        scope.launch {
            loadStoredState()
        }
    }

    fun currentToken(): String? = token

    fun currentServer(): HttpUrl? = server

    fun currentLocalAddress(): InetAddress? = localAddress

    suspend fun setServer(address: HttpUrl, localAddress: InetAddress?) {
        val previous = server
        val changed = previous == null || !sameOrigin(previous, address)
        if (changed) {
            token = null
            runCatching { store.clearSession() }
        }
        server = address
        this.localAddress = localAddress
        runCatching { store.saveServer(address, localAddress) }
        if (changed || _state.value !is SessionState.SignedIn) {
            _state.value = SessionState.SignedOut(address)
        }
    }

    suspend fun login(username: String, password: String): LoginResult {
        val origin = server ?: return LoginResult.Failure(null)
        val api = apiFactory(origin)
        val login = api.login(
            LoginRequest(
                username = username,
                password = password,
                deviceName = deviceName,
                client = client
            )
        )
        return when (login) {
            is ApiResult.Success -> {
                token = login.value.token
                val me = api.me()
                when (me) {
                    is ApiResult.Success -> {
                        val saved = runCatching {
                            saveSession(
                                origin = origin,
                                token = login.value.token,
                                expiresAt = login.value.expiresAt,
                                sessionId = login.value.sessionId,
                                username = me.value.username,
                                role = me.value.role
                            )
                        }
                        if (saved.isSuccess) {
                            LoginResult.Success
                        } else {
                            token = null
                            LoginResult.Failure(saved.exceptionOrNull()?.message)
                        }
                    }
                    is ApiResult.HttpError -> {
                        token = null
                        if (me.code == 401) LoginResult.InvalidCredentials
                        else LoginResult.Failure(me.message)
                    }
                    is ApiResult.NetworkError -> {
                        token = null
                        LoginResult.Failure(me.cause.message)
                    }
                }
            }
            is ApiResult.HttpError -> when (login.code) {
                401 -> LoginResult.InvalidCredentials
                429 -> LoginResult.RateLimited(login.retryAfterSeconds)
                else -> LoginResult.Failure(login.message)
            }
            is ApiResult.NetworkError -> LoginResult.Failure(login.cause.message)
        }
    }

    suspend fun logout() {
        val origin = server
        if (origin != null) {
            runCatching { apiFactory(origin).logout() }
        }
        clearLocalSession()
        _state.value = origin?.let { SessionState.SignedOut(it) } ?: SessionState.NoServer
    }

    fun onUnauthorized() {
        token = null
        _state.value = server?.let { SessionState.SignedOut(it) } ?: SessionState.NoServer
        scope.launch {
            runCatching { store.clearSession() }
        }
    }

    fun startPairing(): Flow<PairingState> {
        val origin = server ?: return flowOf(PairingState.Failed(null))
        return flow {
            val api = apiFactory(origin)
            val created = api.pairCreate(PairCreateRequest(deviceName, client))
            val pair = when (created) {
                is ApiResult.Success -> created.value
                is ApiResult.HttpError -> {
                    emit(PairingState.Failed(created.message))
                    return@flow
                }
                is ApiResult.NetworkError -> {
                    emit(PairingState.Failed(created.cause.message))
                    return@flow
                }
            }
            val expiresAtMillis = clock() + pair.expiresIn * 1000L
            val verificationUrl = origin.newBuilder()
                .addPathSegments(pair.verificationPath.trim('/'))
                .addQueryParameter("codigo", pair.code)
                .build()
                .toString()
            emit(PairingState.Waiting(pair.code, verificationUrl, expiresAtMillis))

            var waitSeconds = pair.interval.coerceAtLeast(1)
            var networkFailures = 0
            while (true) {
                if (clock() >= expiresAtMillis) {
                    emit(PairingState.Expired)
                    return@flow
                }
                delay(waitSeconds * 1000L)
                if (clock() >= expiresAtMillis) {
                    emit(PairingState.Expired)
                    return@flow
                }
                when (val result = api.pairToken(PairTokenRequest(pair.pairId, pair.secret))) {
                    is ApiResult.Success -> {
                        val approved = result.value
                        val saved = runCatching {
                            saveSession(
                                origin = origin,
                                token = approved.token,
                                expiresAt = approved.expiresAt,
                                sessionId = approved.sessionId,
                                username = approved.user.username,
                                role = approved.user.role
                            )
                        }
                        if (saved.isSuccess) {
                            emit(PairingState.Approved(approved.user.username, approved.user.role))
                        } else {
                            emit(PairingState.Failed(saved.exceptionOrNull()?.message))
                        }
                        return@flow
                    }
                    is ApiResult.HttpError -> when (result.code) {
                        410 -> {
                            emit(PairingState.Expired)
                            return@flow
                        }
                        404 -> {
                            emit(PairingState.Failed(result.message))
                            return@flow
                        }
                        428, 429 -> {
                            waitSeconds = maxOf(
                                pair.interval,
                                (result.retryAfterSeconds ?: 0L).toInt()
                            ).coerceAtLeast(1)
                        }
                        else -> {
                            emit(PairingState.Failed(result.message))
                            return@flow
                        }
                    }
                    is ApiResult.NetworkError -> {
                        // Un corte breve de Wi-Fi no debe obligar a empezar con otro código.
                        networkFailures++
                        if (networkFailures >= MAX_POLL_NETWORK_FAILURES) {
                            emit(PairingState.Failed(result.cause.message))
                            return@flow
                        }
                        continue
                    }
                }
                networkFailures = 0
            }
        }
    }

    suspend fun checkServer(address: HttpUrl): ServerCheckResult =
        checkServer(address, apiFactory, appVersion)

    private suspend fun loadStoredState() {
        try {
            val config = store.readConfig()
            server = config.server
            localAddress = config.localAddress
            val currentServer = config.server
            if (currentServer == null) {
                _state.value = SessionState.NoServer
                return
            }
            val stored = store.readSession()
            if (stored != null && sameOrigin(stored.server, currentServer)) {
                token = stored.token
                _state.value = SessionState.SignedIn(currentServer, stored.username, stored.role)
            } else {
                _state.value = SessionState.SignedOut(currentServer)
            }
        } catch (_: Exception) {
            _state.value = SessionState.NoServer
        }
    }

    private suspend fun saveSession(
        origin: HttpUrl,
        token: String,
        expiresAt: String,
        sessionId: Long,
        username: String,
        role: String
    ) {
        store.saveSession(
            server = origin,
            localAddress = localAddress,
            token = token,
            expiresAt = expiresAt,
            sessionId = sessionId,
            username = username,
            role = role
        )
        this.token = token
        _state.value = SessionState.SignedIn(origin, username, role)
    }

    private suspend fun clearLocalSession() {
        token = null
        runCatching { store.clearSession() }
    }

    private companion object {
        const val MAX_POLL_NETWORK_FAILURES = 3
    }

    private fun sameOrigin(first: HttpUrl, second: HttpUrl): Boolean =
        first.scheme.equals(second.scheme, ignoreCase = true) &&
            first.host.equals(second.host, ignoreCase = true) &&
            first.port == second.port
}
