package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.data.PairingState
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class PairingViewModelTest {
    @Test
    fun approvedPairingIsExposed() = runTest {
        val viewModel = PairingViewModel(
            pairing = {
                flowOf(
                    PairingState.Waiting(
                        code = "123456",
                        verificationUrl = "https://cinecat.example.com/vincular?codigo=123456",
                        expiresAtMillis = 5_000
                    ),
                    PairingState.Approved("ana", "admin")
                )
            },
            scope = this
        )

        viewModel.start()
        advanceUntilIdle()

        assertTrue(viewModel.state.value.approved)
        assertEquals(
            PairingState.Approved("ana", "admin"),
            viewModel.state.value.pairing
        )
    }

    @Test
    fun expiredPairingIsExposed() = runTest {
        val viewModel = PairingViewModel(
            pairing = { flowOf(PairingState.Expired) },
            scope = this
        )

        viewModel.start()
        advanceUntilIdle()

        assertEquals(PairingState.Expired, viewModel.state.value.pairing)
        assertEquals(false, viewModel.state.value.approved)
    }

    @Test
    fun restartClearsPreviousState() = runTest {
        val viewModel = PairingViewModel(
            pairing = { flowOf(PairingState.Failed("sin red")) },
            scope = this
        )

        viewModel.start()
        advanceUntilIdle()
        assertEquals(PairingState.Failed("sin red"), viewModel.state.value.pairing)

        viewModel.restart()
        advanceUntilIdle()

        assertEquals(PairingState.Failed("sin red"), viewModel.state.value.pairing)
        assertEquals(false, viewModel.state.value.approved)
    }
}
