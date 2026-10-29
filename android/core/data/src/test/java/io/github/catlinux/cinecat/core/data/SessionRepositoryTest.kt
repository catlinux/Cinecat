package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.createApi
import io.github.catlinux.cinecat.core.api.createOkHttpClient
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.emptyPreferences
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.SocketPolicy
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import kotlinx.coroutines.test.currentTime

@OptIn(ExperimentalCoroutinesApi::class)
class SessionRepositoryTest {
    @get:Rule
    val server = MockWebServer()

    private lateinit var store: SessionStore
    private lateinit var repository: SessionRepository

    @Before
    fun setUp() {
        store = SessionStore(InMemoryPreferencesStore(), FakeTokenCipher())
    }

    @Test
    fun loginStoresSessionAndRole() = runTest {
        val repository = prepareRepository()
        server.enqueue(loginResponse("token-1"))
        server.enqueue(jsonResponse("""{"username":"ana","role":"admin"}"""))

        assertEquals(LoginResult.Success, repository.login("ana", "secret"))
        val state = repository.state.value
        assertTrue(state is SessionState.SignedIn)
        state as SessionState.SignedIn
        assertEquals("ana", state.user)
        assertEquals("admin", state.role)
        assertEquals("token-1", repository.currentToken())
        assertEquals("token-1", store.readSession()?.token)
        assertEquals("/api/v1/auth/login", server.takeRequest().path)
        assertEquals("Bearer token-1", server.takeRequest().getHeader("Authorization"))
    }

    @Test
    fun restoresStoredSession() = runTest {
        val origin = server.url("/")
        store.saveSession(
            server = origin,
            localAddress = null,
            token = "stored-token",
            expiresAt = "2026-10-30T00:00:00Z",
            sessionId = 5,
            username = "ana",
            role = "user"
        )

        val repository = prepareRepository()

        val state = repository.state.value
        assertTrue(state is SessionState.SignedIn)
        state as SessionState.SignedIn
        assertEquals("ana", state.user)
        assertEquals("user", state.role)
        assertEquals("stored-token", repository.currentToken())
    }

    @Test
    fun loginMapsUnauthorizedAndRateLimited() = runTest {
        val repository = prepareRepository()
        server.enqueue(jsonResponse("""{"error":"credenciales"}""").setResponseCode(401))
        assertEquals(LoginResult.InvalidCredentials, repository.login("ana", "bad"))
        assertTrue(repository.state.value is SessionState.SignedOut)
        assertNull(repository.currentToken())

        server.enqueue(
            jsonResponse("""{"error":"demasiado rápido"}""")
                .setResponseCode(429)
                .setHeader("Retry-After", "7")
        )
        assertEquals(LoginResult.RateLimited(7L), repository.login("ana", "secret"))
    }

    @Test
    fun pairingHandlesPendingAndApproval() = runTest {
        val repository = prepareRepository()
        server.enqueue(pairCreateResponse())
        server.enqueue(MockResponse().setResponseCode(428))
        server.enqueue(MockResponse().setResponseCode(428))
        server.enqueue(pairTokenResponse())

        val states = repository.startPairing().take(2).toList()
        val waiting = states[0] as PairingState.Waiting
        assertEquals("ABCD-EFGH", waiting.code)
        assertEquals(
            server.url("/").newBuilder()
                .addPathSegments("vincular")
                .addQueryParameter("codigo", "ABCD-EFGH")
                .build()
                .toString(),
            waiting.verificationUrl
        )
        assertTrue(states[1] is PairingState.Approved)
        assertTrue(repository.state.value is SessionState.SignedIn)
        assertEquals("token-2", repository.currentToken())
    }

    @Test
    fun pairingMapsExpiredAndFailed() = runTest {
        val repository = prepareRepository()
        server.enqueue(pairCreateResponse())
        server.enqueue(MockResponse().setResponseCode(410))
        val expired = repository.startPairing().take(2).toList()
        assertTrue(expired[1] is PairingState.Expired)

        val repository2 = prepareRepository()
        server.enqueue(pairCreateResponse())
        server.enqueue(jsonResponse("""{"error":"no válido"}""").setResponseCode(404))
        val failed = repository2.startPairing().take(2).toList()
        assertTrue(failed[1] is PairingState.Failed)
    }

    @Test
    fun pairingHonoursRetryAfter() = runTest {
        val repository = prepareRepository()
        server.enqueue(pairCreateResponse())
        server.enqueue(
            jsonResponse("""{"error":"demasiado rápido"}""")
                .setResponseCode(429)
                .setHeader("Retry-After", "5")
        )
        server.enqueue(pairTokenResponse())
        val states = repository.startPairing().take(2).toList()

        assertTrue(states[1] is PairingState.Approved)
        assertTrue(testScheduler.currentTime >= 6_000L)
    }

    @Test
    fun pairingSurvivesBriefNetworkLossButNotAPersistentOne() = runTest {
        val repository = prepareRepository()
        server.enqueue(pairCreateResponse())
        server.enqueue(MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AT_START))
        server.enqueue(pairTokenResponse())
        val recovered = repository.startPairing().take(2).toList()
        assertTrue(recovered[1] is PairingState.Approved)

        val repository2 = prepareRepository()
        server.enqueue(pairCreateResponse())
        repeat(10) { server.enqueue(MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AT_START)) }
        val failed = repository2.startPairing().take(2).toList()
        assertTrue(failed[1] is PairingState.Failed)
    }

    @Test
    fun logoutClearsSessionEvenIfServerFails() = runTest {
        val repository = prepareRepository()
        login(repository)
        server.enqueue(jsonResponse("""{"error":"fallo"}""").setResponseCode(500))

        repository.logout()

        assertTrue(repository.state.value is SessionState.SignedOut)
        assertNull(repository.currentToken())
        assertNull(store.readSession())
    }

    @Test
    fun unauthorizedClearsSession() = runTest {
        val repository = prepareRepository()
        login(repository)

        repository.onUnauthorized()
        advanceUntilIdle()

        assertTrue(repository.state.value is SessionState.SignedOut)
        assertNull(repository.currentToken())
        assertNull(store.readSession())
    }

    @Test
    fun changingServerClosesSession() = runTest {
        val repository = prepareRepository()
        login(repository)
        val other = MockWebServer()
        other.start()
        try {
            repository.setServer(other.url("/"), null)
            assertTrue(repository.state.value is SessionState.SignedOut)
            assertNull(repository.currentToken())
            assertNull(store.readSession())
        } finally {
            other.shutdown()
        }
    }

    private suspend fun TestScope.prepareRepository(): SessionRepository {
        lateinit var created: SessionRepository
        val client = createOkHttpClient(
            serverOrigin = { server.url("/") },
            token = { created.currentToken() },
            onUnauthorized = { created.onUnauthorized() },
            version = "0.30.0"
        )
        created = SessionRepository(
            store = store,
            apiFactory = { url -> createApi(url, client) },
            deviceName = "TV de pruebas",
            client = "androidtv",
            appVersion = "0.30.0",
            scope = this,
            clock = { testScheduler.currentTime }
        )
        repository = created
        created.state.first { it !is SessionState.Loading }
        created.setServer(server.url("/"), null)
        return created
    }

    private suspend fun TestScope.login(repository: SessionRepository) {
        server.enqueue(loginResponse("token-1"))
        server.enqueue(jsonResponse("""{"username":"ana","role":"admin"}"""))
        assertEquals(LoginResult.Success, repository.login("ana", "secret"))
    }

    private fun loginResponse(token: String): MockResponse =
        jsonResponse(
            """{"token":"$token","expires_at":"2026-10-30T00:00:00Z","session_id":1}"""
        )

    private fun pairCreateResponse(): MockResponse =
        jsonResponse(
            """{"pair_id":"pair-1","secret":"secret-1","code":"ABCD-EFGH","verification_path":"/vincular","expires_in":600,"interval":1}"""
        )

    private fun pairTokenResponse(): MockResponse =
        jsonResponse(
            """{"token":"token-2","expires_at":"2026-10-30T00:00:00Z","session_id":2,"user":{"username":"ana","role":"admin"}}"""
        )

    private fun jsonResponse(body: String): MockResponse =
        MockResponse()
            .setResponseCode(200)
            .setHeader("Content-Type", "application/json")
            .setBody(body)

    private class FakeTokenCipher : TokenCipher {
        override fun encrypt(data: ByteArray): ByteArray = data.reversedArray()
        override fun decrypt(data: ByteArray): ByteArray = data.reversedArray()
    }

    private class InMemoryPreferencesStore : DataStore<Preferences> {
        private val value = MutableStateFlow<Preferences>(emptyPreferences())
        override val data = value

        override suspend fun updateData(
            transform: suspend (t: Preferences) -> Preferences
        ): Preferences {
            val next = transform(value.value)
            value.value = next
            return next
        }
    }
}
