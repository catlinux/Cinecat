package io.github.catlinux.cinecat.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Library(
    val id: Long,
    val name: String = "",
    val kind: String = "",
    @SerialName("item_count") val itemCount: Long = 0
)

@Serializable
data class LibrariesResponse(
    val libraries: List<Library> = emptyList()
)

@Serializable
data class PhotoItem(
    val id: Long,
    val name: String = "",
    val width: Int = 0,
    val height: Int = 0,
    @SerialName("taken_at") val takenAt: Long? = null,
    val size: Long = 0
)

@Serializable
data class PhotosResponse(
    val folders: List<String> = emptyList(),
    val items: List<PhotoItem> = emptyList(),
    val total: Long = 0
)
