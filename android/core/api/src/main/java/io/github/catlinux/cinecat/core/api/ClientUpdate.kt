package io.github.catlinux.cinecat.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ClientUpdate(
    val version: String = "",
    @SerialName("version_code") val versionCode: Long = 0,
    val url: String = "",
    val sha256: String = "",
    val notes: String = ""
)
