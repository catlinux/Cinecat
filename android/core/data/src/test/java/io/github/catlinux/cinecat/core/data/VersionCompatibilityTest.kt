package io.github.catlinux.cinecat.core.data

import io.github.catlinux.cinecat.core.api.ServerVersion
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class VersionCompatibilityTest {
    @Test
    fun comparesVersions() {
        assertEquals(0, compareVersions("0.30.0", "0.30.0"))
        assertTrue(compareVersions("0.30.1", "0.30.0") > 0)
        assertTrue(compareVersions("0.29.9", "0.30.0") < 0)
        assertTrue(compareVersions("1.0.0", "0.99.99") > 0)
        assertEquals(0, compareVersions("0.35.1-debug", "0.35.1"))
        assertEquals(0, compareVersions("0.35.1+abc", "0.35.1"))
    }

    @Test
    fun checksAllCompatibilityCases() {
        assertEquals(
            Compatibility.ServerTooOld,
            checkCompatibility(version(apiLevel = 0, minClient = "0.0.0"), "0.30.0")
        )
        assertEquals(
            Compatibility.ApiIncompatible,
            checkCompatibility(version(apiLevel = 2, minClient = "0.0.0"), "0.30.0")
        )
        assertEquals(
            Compatibility.AppTooOld("0.31.0"),
            checkCompatibility(version(apiLevel = 1, minClient = "0.31.0"), "0.30.0")
        )
        assertEquals(
            Compatibility.Compatible,
            checkCompatibility(version(apiLevel = 1, minClient = "0.30.0"), "0.30.0")
        )
    }

    private fun version(apiLevel: Int, minClient: String): ServerVersion =
        ServerVersion(
            server = "0.30.0",
            apiLevel = apiLevel,
            minClient = minClient
        )
}
