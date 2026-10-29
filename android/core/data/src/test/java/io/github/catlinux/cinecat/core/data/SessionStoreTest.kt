package io.github.catlinux.cinecat.core.data

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.core.Preferences
import java.io.File
import java.net.InetAddress
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.runBlocking
import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder

class SessionStoreTest {
    @get:Rule
    val temporaryFolder = TemporaryFolder()

    private lateinit var scope: CoroutineScope
    private lateinit var dataStore: DataStore<Preferences>
    private lateinit var file: File
    private lateinit var cipher: FakeTokenCipher
    private lateinit var store: SessionStore

    @Before
    fun setUp() {
        scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
        file = File(temporaryFolder.root, "session.preferences_pb")
        dataStore = PreferenceDataStoreFactory.create(
            scope = scope,
            produceFile = { file }
        )
        cipher = FakeTokenCipher()
        store = SessionStore(dataStore, cipher)
    }

    @After
    fun tearDown() {
        scope.cancel()
    }

    @Test
    fun savesAndReadsServerConfigAndSession() = runBlocking {
        val server = "https://cinecat.example/".toHttpUrl()
        val local = InetAddress.getByAddress(byteArrayOf(192.toByte(), 168.toByte(), 1, 10))
        store.saveSession(server, local, TOKEN, "2026-10-30T00:00:00Z", 9, "ana", "admin")

        val config = store.readConfig()
        assertEquals(server, config.server)
        assertEquals(local, config.localAddress)

        val session = store.readSession()
        assertNotNull(session)
        assertEquals(server, session?.server)
        assertEquals(local, session?.localAddress)
        assertEquals(TOKEN, session?.token)
        assertEquals(9L, session?.sessionId)
        assertEquals("ana", session?.username)
        assertEquals("admin", session?.role)
    }

    @Test
    fun encryptedFileDoesNotContainPlainToken() = runBlocking {
        val server = "https://cinecat.example/".toHttpUrl()
        store.saveSession(server, null, TOKEN, "2026-10-30T00:00:00Z", 1, "ana", "user")

        val contents = file.readBytes().toString(Charsets.ISO_8859_1)
        assertFalse(contents.contains(TOKEN))
    }

    @Test
    fun failedDecryptionClearsSessionAndKeepsServer() = runBlocking {
        val server = "https://cinecat.example/".toHttpUrl()
        store.saveSession(server, null, TOKEN, "2026-10-30T00:00:00Z", 1, "ana", "user")
        cipher.failDecrypt = true

        assertNull(store.readSession())
        assertNull(store.readSession())
        assertEquals(server, store.readConfig().server)
    }

    private class FakeTokenCipher : TokenCipher {
        var failDecrypt = false

        override fun encrypt(data: ByteArray): ByteArray = data.reversedArray()

        override fun decrypt(data: ByteArray): ByteArray {
            if (failDecrypt) {
                throw IllegalStateException("No se puede descifrar")
            }
            return data.reversedArray()
        }
    }

    private companion object {
        const val TOKEN = "super-secret-token-123"
    }
}
