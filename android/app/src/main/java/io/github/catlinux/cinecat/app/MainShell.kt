package io.github.catlinux.cinecat.app

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountCircle
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationRail
import androidx.compose.material3.NavigationRailItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.toRoute
import io.github.catlinux.cinecat.core.data.SessionState
import kotlinx.serialization.Serializable

@Serializable
data object Portal

@Serializable
data object Movies

@Serializable
data object Series

@Serializable
data object Collections

@Serializable
data object Search

@Serializable
data object Account

@Serializable
data class MovieDetail(val id: Long)

@Serializable
data class SeriesDetail(val id: Long)

@Serializable
data class CollectionDetail(val id: Long)

@Serializable
data class Player(val itemId: Long, val startSeconds: Double = 0.0)

private data class MainTab(
    @StringRes val labelRes: Int,
    val route: Any,
    val icon: ImageVector? = null,
    @DrawableRes val iconRes: Int? = null
)

@Composable
fun MainShell(
    container: AppContainer,
    session: SessionState.SignedIn,
    isTelevision: Boolean,
    onChangeServer: () -> Unit
) {
    val navController = rememberNavController()
    val tabs = remember { mainTabs() }
    val useRail = useNavigationRail(LocalConfiguration.current.screenWidthDp)
    val currentEntry by navController.currentBackStackEntryAsState()
    val currentDestination = currentEntry?.destination
    val isTopLevel = currentDestination != null &&
        tabs.any { tab -> currentDestination.hasRoute(tab.route::class) }
    val isPlayer = currentDestination?.hasRoute(Player::class) == true

    fun navigateTopLevel(route: Any) {
        if (currentDestination?.hasRoute(route::class) == true) {
            return
        }
        if (route is Portal) {
            navController.popBackStack(Portal, false)
        } else {
            navController.navigate(route) {
                popUpTo(Portal) { saveState = true }
                launchSingleTop = true
                restoreState = true
            }
        }
    }

    Scaffold(
        modifier = Modifier.fillMaxSize(),
        contentWindowInsets = if (isPlayer) WindowInsets(0, 0, 0, 0) else WindowInsets.safeDrawing,
        bottomBar = {
            if (!useRail && isTopLevel) {
                NavigationBar {
                    tabs.forEach { tab ->
                        NavigationBarItem(
                            selected = currentDestination.hasRoute(tab.route::class),
                            onClick = { navigateTopLevel(tab.route) },
                            icon = { TabIcon(tab) },
                            label = { Text(stringResource(tab.labelRes)) }
                        )
                    }
                }
            }
        }
    ) { innerPadding ->
        Row(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            if (useRail && isTopLevel) {
                NavigationRail {
                    Spacer(Modifier.height(8.dp))
                    tabs.forEach { tab ->
                        NavigationRailItem(
                            selected = currentDestination.hasRoute(tab.route::class),
                            onClick = { navigateTopLevel(tab.route) },
                            modifier = Modifier.focusRing(RoundedCornerShape(16.dp)),
                            icon = { TabIcon(tab) },
                            label = { Text(stringResource(tab.labelRes)) },
                            alwaysShowLabel = true
                        )
                    }
                    // En el riel la cuenta va aquí: en la TV el icono de la cabecera solo se
                    // alcanzaba con el mando recorriendo la fila entera hacia la derecha.
                    NavigationRailItem(
                        selected = false,
                        onClick = { navController.navigate(Account) },
                        modifier = Modifier.focusRing(RoundedCornerShape(16.dp)),
                        icon = { Icon(Icons.Filled.AccountCircle, contentDescription = null) },
                        label = { Text(stringResource(R.string.account_title)) },
                        alwaysShowLabel = true
                    )
                }
            }
            Box(modifier = Modifier.weight(1f).fillMaxHeight()) {
                NavHost(
                    navController = navController,
                    startDestination = Portal,
                    modifier = Modifier.fillMaxSize()
                ) {
                    composable<Portal> {
                        val portalViewModel: PortalViewModel = viewModel {
                            PortalViewModel(container.libraryRepository, container.appScope)
                        }
                        PortalScreen(
                            viewModel = portalViewModel,
                            isTelevision = isTelevision,
                            onOpenAccount = if (useRail) null else ({ navController.navigate(Account) }),
                            onOpenMovie = { navController.navigate(MovieDetail(it)) },
                            onOpenSeries = { navController.navigate(SeriesDetail(it)) }
                        )
                    }
                    composable<Movies> {
                        val moviesViewModel: MoviesViewModel = viewModel {
                            MoviesViewModel(container.libraryRepository, container.appScope)
                        }
                        MoviesScreen(
                            viewModel = moviesViewModel,
                            onOpenMovie = { navController.navigate(MovieDetail(it)) }
                        )
                    }
                    composable<Series> {
                        val seriesViewModel: SeriesViewModel = viewModel {
                            SeriesViewModel(container.libraryRepository, container.appScope)
                        }
                        SeriesScreen(
                            viewModel = seriesViewModel,
                            onOpenSeries = { navController.navigate(SeriesDetail(it)) }
                        )
                    }
                    composable<Collections> {
                        val collectionsViewModel: CollectionsViewModel = viewModel {
                            CollectionsViewModel(container.libraryRepository, container.appScope)
                        }
                        CollectionsScreen(
                            viewModel = collectionsViewModel,
                            onOpenCollection = {
                                navController.navigate(CollectionDetail(it))
                            }
                        )
                    }
                    composable<Search> {
                        val searchViewModel: SearchViewModel = viewModel {
                            SearchViewModel(container.libraryRepository, container.appScope)
                        }
                        SearchScreen(
                            viewModel = searchViewModel,
                            onOpenMovie = { navController.navigate(MovieDetail(it)) },
                            onOpenSeries = { navController.navigate(SeriesDetail(it)) }
                        )
                    }
                    composable<Account> {
                        AccountScreen(
                            repository = container.sessionRepository,
                            session = session,
                            updater = container.appUpdater,
                            isTelevision = isTelevision,
                            onBack = { navController.popBackStack() },
                            onChangeServer = onChangeServer
                        )
                    }
                    composable<MovieDetail> {
                        val route = it.toRoute<MovieDetail>()
                        val movieViewModel: MovieDetailViewModel = viewModel {
                            MovieDetailViewModel(
                                id = route.id,
                                repository = container.libraryRepository,
                                scope = container.appScope
                            )
                        }
                        MovieDetailScreen(
                            viewModel = movieViewModel,
                            onBack = { navController.popBackStack() },
                            onPlay = { id, position ->
                                navController.navigate(Player(id, position))
                            },
                            onOpenMovie = { id -> navController.navigate(MovieDetail(id)) },
                            onOpenCollection = { id ->
                                navController.navigate(CollectionDetail(id))
                            }
                        )
                    }
                    composable<SeriesDetail> {
                        val route = it.toRoute<SeriesDetail>()
                        val seriesViewModel: SeriesDetailViewModel = viewModel {
                            SeriesDetailViewModel(
                                id = route.id,
                                repository = container.libraryRepository,
                                scope = container.appScope
                            )
                        }
                        SeriesDetailScreen(
                            viewModel = seriesViewModel,
                            onBack = { navController.popBackStack() },
                            onPlayEpisode = { id, position ->
                                navController.navigate(Player(id, position))
                            },
                            onOpenSeries = { id -> navController.navigate(SeriesDetail(id)) }
                        )
                    }
                    composable<Player> {
                        val route = it.toRoute<Player>()
                        val playerViewModel: PlayerViewModel = viewModel {
                            PlayerViewModel(
                                itemId = route.itemId,
                                startSeconds = route.startSeconds,
                                repository = container.playbackRepository,
                                caps = container.caps,
                                scope = container.appScope,
                                serverUrl = { container.sessionRepository.currentServer() }
                            )
                        }
                        PlayerScreen(
                            viewModel = playerViewModel,
                            client = container.client,
                            isTelevision = isTelevision,
                            onBack = { navController.popBackStack() }
                        )
                    }
                    composable<CollectionDetail> {
                        val route = it.toRoute<CollectionDetail>()
                        val collectionViewModel: CollectionDetailViewModel = viewModel {
                            CollectionDetailViewModel(
                                id = route.id,
                                repository = container.libraryRepository,
                                scope = container.appScope
                            )
                        }
                        CollectionDetailScreen(
                            viewModel = collectionViewModel,
                            onBack = { navController.popBackStack() },
                            onOpenMovie = { id -> navController.navigate(MovieDetail(id)) }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun TabIcon(tab: MainTab) {
    val icon = tab.icon
    if (icon != null) {
        Icon(imageVector = icon, contentDescription = null)
    } else {
        Icon(painter = painterResource(tab.iconRes!!), contentDescription = null)
    }
}

private fun mainTabs(): List<MainTab> = listOf(
    MainTab(R.string.nav_portal, Portal, icon = Icons.Filled.Home),
    MainTab(R.string.nav_movies, Movies, iconRes = R.drawable.ic_nav_movies),
    MainTab(R.string.nav_series, Series, iconRes = R.drawable.ic_nav_series),
    MainTab(R.string.nav_collections, Collections, iconRes = R.drawable.ic_nav_collections),
    MainTab(R.string.nav_search, Search, icon = Icons.Filled.Search)
)
