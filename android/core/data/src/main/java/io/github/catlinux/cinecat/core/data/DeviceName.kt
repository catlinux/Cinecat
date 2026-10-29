package io.github.catlinux.cinecat.core.data

import android.content.Context
import android.os.Build
import android.provider.Settings

fun deviceName(context: Context): String {
    val globalName = runCatching {
        Settings.Global.getString(context.contentResolver, Settings.Global.DEVICE_NAME)
    }.getOrNull()?.trim()
    val fallback = "${Build.MANUFACTURER} ${Build.MODEL}".trim()
    return (if (globalName.isNullOrEmpty()) fallback else globalName)
        .take(64)
        .ifBlank { "Android" }
}
