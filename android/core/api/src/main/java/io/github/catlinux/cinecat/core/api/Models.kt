package io.github.catlinux.cinecat.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ServerVersion(
    val server: String,
    val commit: String = "",
    @SerialName("built_at") val builtAt: String = "",
    // Los servidores anteriores a 0.30.0 no envían estos campos: apiLevel 0 los identifica.
    @SerialName("api_level") val apiLevel: Int = 0,
    @SerialName("min_client") val minClient: String = "0.0.0"
)

@Serializable
data class LoginRequest(
    val username: String,
    val password: String,
    @SerialName("device_name") val deviceName: String? = null,
    val client: String? = null
)

@Serializable
data class LoginResponse(
    val token: String,
    @SerialName("expires_at") val expiresAt: String,
    @SerialName("session_id") val sessionId: Long
)

@Serializable
data class CurrentUserResponse(
    val username: String,
    val role: String,
    @SerialName("all_libraries") val allLibraries: Boolean? = null,
    @SerialName("library_ids") val libraryIds: List<Long>? = null
)

@Serializable
data class PairCreateRequest(
    @SerialName("device_name") val deviceName: String,
    val client: String
)

@Serializable
data class PairCreateResponse(
    @SerialName("pair_id") val pairId: String,
    val secret: String,
    val code: String,
    @SerialName("verification_path") val verificationPath: String,
    @SerialName("expires_in") val expiresIn: Int,
    val interval: Int
)

@Serializable
data class PairTokenRequest(
    @SerialName("pair_id") val pairId: String,
    val secret: String
)

@Serializable
data class PairTokenResponse(
    val token: String,
    @SerialName("expires_at") val expiresAt: String,
    @SerialName("session_id") val sessionId: Long,
    val user: PairUser
)

@Serializable
data class PairUser(
    val username: String,
    val role: String
)

@Serializable
data class ErrorResponse(
    val error: String
)
