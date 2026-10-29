package io.github.catlinux.cinecat.core.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AddressTest {
    @Test
    fun parsesServerOrigins() {
        assertEquals(
            "https://cinecat.example/",
            ServerAddress.parse(" cinecat.example ").getOrThrow().toString()
        )
        assertEquals(
            "https://cinecat.example/",
            ServerAddress.parse("https://Cinecat.Example/").getOrThrow().toString()
        )
        assertEquals(
            "https://cinecat.example:8443/",
            ServerAddress.parse("https://cinecat.example:8443").getOrThrow().toString()
        )
    }

    @Test
    fun rejectsNonOriginServers() {
        listOf(
            "",
            "http://cinecat.example",
            "ftp://cinecat.example",
            "https://user:pass@cinecat.example",
            "https://cinecat.example/web",
            "https://cinecat.example?x=1",
            "https://cinecat.example#fragment",
            "https://cinecat.example:bad"
        ).forEach { input ->
            assertTrue("debería rechazar $input", ServerAddress.parse(input).isFailure)
        }
    }

    @Test
    fun parsesPrivateLocalAddresses() {
        assertNull(LocalAddress.parse("  ").getOrThrow())
        listOf(
            "10.0.0.1",
            "172.16.0.1",
            "172.31.255.255",
            "192.168.1.10",
            "fc00::1",
            "fd00::1",
            "fe80::1"
        ).forEach { input ->
            assertNotNull("debería aceptar $input", LocalAddress.parse(input).getOrThrow())
        }
        assertEquals(
            "192.168.1.10",
            LocalAddress.parse("192.168.1.10").getOrThrow()?.hostAddress
        )
    }

    @Test
    fun rejectsNonPrivateOrNonLiteralAddresses() {
        listOf(
            "8.8.8.8",
            "172.15.0.1",
            "172.32.0.1",
            "192.169.0.1",
            "11.0.0.1",
            "169.254.1.1",
            "router.local",
            "localhost",
            "::1",
            "2001:db8::1",
            "fec0::1",
            "fe80::1%eth0"
        ).forEach { input ->
            assertTrue("debería rechazar $input", LocalAddress.parse(input).isFailure)
        }
    }

    @Test
    fun rejectsIpv4NamesWithoutResolvingThem() {
        assertTrue(LocalAddress.parse("private.example").isFailure)
    }
}
