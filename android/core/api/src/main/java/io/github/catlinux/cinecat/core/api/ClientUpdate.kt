package io.github.catlinux.cinecat.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Resultado de preguntar al servidor: la actualización si la hay o, si no, el
 * motivo que da el servidor en X-Cinecat-Update-Reason (disabled, current,
 * unavailable…; null en servidores anteriores a 0.37.1).
 */
data class ClientUpdateCheck(
    val update: ClientUpdate?,
    val reason: String? = null
)

@Serializable
data class ClientUpdate(
    val version: String = "",
    @SerialName("version_code") val versionCode: Long = 0,
    val url: String = "",
    val sha256: String = "",
    val notes: String = ""
)
