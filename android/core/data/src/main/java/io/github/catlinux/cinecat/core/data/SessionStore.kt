package io.github.catlinux.cinecat.core.data

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import java.net.InetAddress
import java.util.Base64
import kotlinx.coroutines.flow.first
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

class SessionStore(
    private val dataStore: DataStore<Preferences>,
    private val cipher: TokenCipher
) {
    data class ServerConfig(
        val server: HttpUrl?,
        val localAddress: InetAddress?
    )

    data class StoredSession(
        val server: HttpUrl,
        val localAddress: InetAddress?,
        val token: String,
        val expiresAt: String,
        val sessionId: Long,
        val username: String,
        val role: String
    )

    suspend fun readConfig(): ServerConfig {
        val preferences = dataStore.data.first()
        val server = preferences[SERVER]?.toHttpUrlOrNull()
        val localAddress = preferences[LOCAL_ADDRESS]
            ?.let { LocalAddress.parse(it).getOrNull() }
        return ServerConfig(server, localAddress)
    }

    suspend fun readSession(): StoredSession? {
        val preferences = dataStore.data.first()
        val server = preferences[SERVER]?.toHttpUrlOrNull() ?: return null
        val localAddress = preferences[LOCAL_ADDRESS]
            ?.let { LocalAddress.parse(it).getOrNull() }
        val encrypted = preferences[TOKEN] ?: return null
        val expiresAt = preferences[EXPIRES_AT] ?: return null
        val sessionId = preferences[SESSION_ID] ?: return null
        val username = preferences[USERNAME] ?: return null
        val role = preferences[ROLE] ?: return null
        val token = runCatching {
            String(cipher.decrypt(Base64.getDecoder().decode(encrypted)), Charsets.UTF_8)
        }.getOrNull()
        if (token.isNullOrEmpty()) {
            runCatching { clearSession() }
            return null
        }
        return StoredSession(server, localAddress, token, expiresAt, sessionId, username, role)
    }

    suspend fun saveServer(server: HttpUrl, localAddress: InetAddress?) {
        dataStore.edit { preferences ->
            preferences[SERVER] = server.toString()
            if (localAddress == null) {
                preferences.remove(LOCAL_ADDRESS)
            } else {
                preferences[LOCAL_ADDRESS] = localAddress.hostAddress.orEmpty()
            }
        }
    }

    suspend fun saveSession(
        server: HttpUrl,
        localAddress: InetAddress?,
        token: String,
        expiresAt: String,
        sessionId: Long,
        username: String,
        role: String
    ) {
        val encrypted = Base64.getEncoder().encodeToString(
            cipher.encrypt(token.toByteArray(Charsets.UTF_8))
        )
        dataStore.edit { preferences ->
            preferences[SERVER] = server.toString()
            if (localAddress == null) {
                preferences.remove(LOCAL_ADDRESS)
            } else {
                preferences[LOCAL_ADDRESS] = localAddress.hostAddress.orEmpty()
            }
            preferences[TOKEN] = encrypted
            preferences[EXPIRES_AT] = expiresAt
            preferences[SESSION_ID] = sessionId
            preferences[USERNAME] = username
            preferences[ROLE] = role
        }
    }

    suspend fun clearSession() {
        dataStore.edit { preferences ->
            preferences.remove(TOKEN)
            preferences.remove(EXPIRES_AT)
            preferences.remove(SESSION_ID)
            preferences.remove(USERNAME)
            preferences.remove(ROLE)
        }
    }

    private companion object {
        val SERVER = stringPreferencesKey("server")
        val LOCAL_ADDRESS = stringPreferencesKey("local_address")
        val TOKEN = stringPreferencesKey("session_token")
        val EXPIRES_AT = stringPreferencesKey("session_expires_at")
        val SESSION_ID = longPreferencesKey("session_id")
        val USERNAME = stringPreferencesKey("username")
        val ROLE = stringPreferencesKey("role")
    }
}
