package io.github.catlinux.cinecat.app

import io.github.catlinux.cinecat.core.data.LoginResult
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LoginViewModelTest {
    @Test
    fun successfulLoginMarksSessionAsSignedIn() = runTest {
        val viewModel = LoginViewModel(
            login = { username, password ->
                assertEquals("ana", username)
                assertEquals("secreta", password)
                LoginResult.Success
            },
            scope = this
        )
        viewModel.onUsernameChanged("ana")
        viewModel.onPasswordChanged("secreta")

        viewModel.login()
        advanceUntilIdle()

        assertTrue(viewModel.state.value.signedIn)
        assertFalse(viewModel.state.value.submitting)
        assertEquals(null, viewModel.state.value.error)
    }

    @Test
    fun invalidCredentialsShowError() = runTest {
        val viewModel = LoginViewModel(
            login = { _, _ -> LoginResult.InvalidCredentials },
            scope = this
        )
        viewModel.onUsernameChanged("ana")
        viewModel.onPasswordChanged("incorrecta")

        viewModel.login()
        advanceUntilIdle()

        assertEquals(LoginError.InvalidCredentials, viewModel.state.value.error)
        assertFalse(viewModel.state.value.signedIn)
    }

    @Test
    fun rateLimitStartsCountdownAndDisablesSubmit() = runTest {
        val viewModel = LoginViewModel(
            login = { _, _ -> LoginResult.RateLimited(2) },
            scope = this
        )
        viewModel.onUsernameChanged("ana")
        viewModel.onPasswordChanged("secreta")

        viewModel.login()
        runCurrent()

        assertEquals(LoginError.TooManyAttempts, viewModel.state.value.error)
        assertEquals(2, viewModel.state.value.retrySeconds)
        assertFalse(viewModel.state.value.canSubmit)

        advanceTimeBy(1_001)
        runCurrent()
        assertEquals(1, viewModel.state.value.retrySeconds)

        advanceTimeBy(1_001)
        runCurrent()
        assertEquals(0, viewModel.state.value.retrySeconds)
        assertEquals(null, viewModel.state.value.error)
        assertTrue(viewModel.state.value.canSubmit)
    }

    @Test
    fun networkFailureIsReported() = runTest {
        val viewModel = LoginViewModel(
            login = { _, _ -> LoginResult.Failure("sin red") },
            scope = this
        )
        viewModel.onUsernameChanged("ana")
        viewModel.onPasswordChanged("secreta")

        viewModel.login()
        advanceUntilIdle()

        assertEquals(LoginError.Network, viewModel.state.value.error)
    }

    @Test
    fun passwordVisibilityCanBeToggled() = runTest {
        val viewModel = LoginViewModel(
            login = { _, _ -> LoginResult.Success },
            scope = this
        )

        viewModel.togglePasswordVisibility()

        assertTrue(viewModel.state.value.passwordVisible)
    }
}
