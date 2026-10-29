package io.github.catlinux.cinecat.app

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.collectAsState
import io.github.catlinux.cinecat.core.data.SessionState

private enum class AppScreen {
    Loading,
    Server,
    Login,
    Pairing,
    Home
}

@Composable
fun CinecatApp(container: AppContainer, isTelevision: Boolean) {
    val session by container.sessionRepository.state.collectAsState()
    var manualScreen by rememberSaveable { mutableStateOf<AppScreen?>(null) }

    val automaticScreen = when (session) {
        SessionState.Loading -> AppScreen.Loading
        SessionState.NoServer -> AppScreen.Server
        is SessionState.SignedOut -> if (isTelevision) AppScreen.Pairing else AppScreen.Login
        is SessionState.SignedIn -> AppScreen.Home
    }
    val screen = manualScreen ?: automaticScreen

    BackHandler(enabled = screen == AppScreen.Login || screen == AppScreen.Pairing) {
        manualScreen = AppScreen.Server
    }

    when (screen) {
        AppScreen.Loading -> LoadingScreen()
        AppScreen.Server -> ServerScreen(
            repository = container.sessionRepository,
            isTelevision = isTelevision,
            scope = container.appScope,
            onSaved = { manualScreen = null }
        )
        AppScreen.Login -> LoginScreen(
            repository = container.sessionRepository,
            scope = container.appScope,
            onUseCode = { manualScreen = AppScreen.Pairing },
            onSignedIn = { manualScreen = null }
        )
        AppScreen.Pairing -> PairingScreen(
            repository = container.sessionRepository,
            isTelevision = isTelevision,
            scope = container.appScope,
            onApproved = { manualScreen = null }
        )
        AppScreen.Home -> MainShell(
            container = container,
            session = session as SessionState.SignedIn,
            isTelevision = isTelevision,
            onChangeServer = { manualScreen = AppScreen.Server }
        )
    }
}
