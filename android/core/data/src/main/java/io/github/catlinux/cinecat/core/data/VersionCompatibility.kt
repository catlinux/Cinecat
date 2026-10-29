package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ServerVersion

sealed interface Compatibility {
    data object Compatible : Compatibility
    data object ServerTooOld : Compatibility
    data object ApiIncompatible : Compatibility
    data class AppTooOld(val minClient: String) : Compatibility
}

fun compareVersions(a: String, b: String): Int {
    val left = parseVersion(a)
    val right = parseVersion(b)
    for (index in 0..2) {
        if (left[index] != right[index]) {
            return left[index].compareTo(right[index])
        }
    }
    return 0
}

fun checkCompatibility(version: ServerVersion, appVersion: String): Compatibility {
    if (version.apiLevel == 0) {
        return Compatibility.ServerTooOld
    }
    if (version.apiLevel != 1) {
        return Compatibility.ApiIncompatible
    }
    if (compareVersions(appVersion, version.minClient) < 0) {
        return Compatibility.AppTooOld(version.minClient)
    }
    return Compatibility.Compatible
}

// Los sufijos SemVer (el "-debug" de la versión de pruebas) no cuentan para la compatibilidad.
private fun parseVersion(value: String): List<Int> {
    val parts = value.trim().substringBefore('-').substringBefore('+').split(".")
    require(parts.size == 3 && parts.all { it.isNotEmpty() && it.all(Char::isDigit) }) {
        "La versión debe tener el formato x.y.z"
    }
    return parts.map { it.toInt() }
}
