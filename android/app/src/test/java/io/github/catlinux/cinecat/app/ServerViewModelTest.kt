package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.api.ServerVersion
import io.github.catlinux.cinecat.core.data.Compatibility
import io.github.catlinux.cinecat.core.data.ServerCheckResult
import java.net.InetAddress
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import okhttp3.HttpUrl
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class ServerViewModelTest {
    @Test
    fun compatibleServerCanBeSaved() = runTest {
        val checked = mutableListOf<HttpUrl>()
        val saved = mutableListOf<Pair<HttpUrl, InetAddress?>>()
        val viewModel = ServerViewModel(
            initialServer = "https://cinecat.example.com/",
            initialLocalAddress = "192.168.1.20",
            checkServer = { address ->
                checked += address
                ServerCheckResult.Compatible(ServerVersion(server = "0.30.0"))
            },
            saveServer = { address, localAddress ->
                saved += address to localAddress
            },
            scope = this
        )

        viewModel.check()
        advanceUntilIdle()

        val checkedState = viewModel.state.value
        assertTrue(checkedState.check is ServerCheckUi.Compatible)
        assertEquals("https://cinecat.example.com/", checked.single().toString())
        val ready = checkedState.ready
        assertNotNull(ready)
        assertEquals("https://cinecat.example.com/", ready!!.origin.toString())
        assertEquals("192.168.1.20", ready.localAddress?.hostAddress)

        viewModel.save()
        advanceUntilIdle()

        assertTrue(viewModel.state.value.saved)
        assertEquals("https://cinecat.example.com/", saved.single().first.toString())
        assertEquals("192.168.1.20", saved.single().second?.hostAddress)
    }

    @Test
    fun invalidAddressIsRejectedBeforeChecking() = runTest {
        var calls = 0
        val viewModel = ServerViewModel(
            initialServer = "http://cinecat.example.com",
            initialLocalAddress = "",
            checkServer = {
                calls++
                ServerCheckResult.Unreachable
            },
            saveServer = { _, _ -> },
            scope = this
        )

        viewModel.check()
        advanceUntilIdle()

        assertEquals(ServerFieldError.Address, viewModel.state.value.fieldError)
        assertEquals(0, calls)
    }

    @Test
    fun publicLocalAddressIsRejected() = runTest {
        val viewModel = ServerViewModel(
            initialServer = "https://cinecat.example.com",
            initialLocalAddress = "8.8.8.8",
            checkServer = { ServerCheckResult.Compatible(ServerVersion("0.30.0")) },
            saveServer = { _, _ -> },
            scope = this
        )

        viewModel.check()
        advanceUntilIdle()

        assertEquals(ServerFieldError.LocalAddress, viewModel.state.value.fieldError)
    }

    @Test
    fun incompatibleResultsAreMappedToMessages() = runTest {
        val version = ServerVersion(
            server = "0.30.0",
            apiLevel = 2,
            minClient = "0.30.0"
        )
        val viewModel = ServerViewModel(
            initialServer = "https://cinecat.example.com",
            initialLocalAddress = "",
            checkServer = {
                ServerCheckResult.Incompatible(
                    version,
                    Compatibility.AppTooOld("0.40.0")
                )
            },
            saveServer = { _, _ -> },
            scope = this
        )

        viewModel.check()
        advanceUntilIdle()

        assertEquals(ServerCheckUi.AppTooOld("0.40.0"), viewModel.state.value.check)
        assertEquals(null, viewModel.state.value.ready)
    }

    @Test
    fun unreachableServerIsReported() = runTest {
        val viewModel = ServerViewModel(
            initialServer = "https://cinecat.example.com",
            initialLocalAddress = "",
            checkServer = { ServerCheckResult.Unreachable },
            saveServer = { _, _ -> },
            scope = this
        )

        viewModel.check()
        advanceUntilIdle()

        assertEquals(ServerCheckUi.Unreachable, viewModel.state.value.check)
    }

    @Test
    fun resetClearsPreviousResult() = runTest {
        val viewModel = ServerViewModel(
            initialServer = "https://cinecat.example.com",
            initialLocalAddress = "",
            checkServer = { ServerCheckResult.Compatible(ServerVersion("0.30.0")) },
            saveServer = { _, _ -> },
            scope = this
        )
        viewModel.check()
        advanceUntilIdle()
        assertTrue(viewModel.state.value.check is ServerCheckUi.Compatible)

        viewModel.reset("https://otro.example.com/", "10.0.0.5")

        assertEquals("https://otro.example.com", viewModel.state.value.server)
        assertEquals("10.0.0.5", viewModel.state.value.localAddress)
        assertEquals(null, viewModel.state.value.check)
        assertEquals(null, viewModel.state.value.ready)
    }
}
