package io.github.catlinux.cinecat.app

import android.annotation.SuppressLint
import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.content.pm.ActivityInfo
import android.content.pm.PackageManager
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.focusable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.media3.common.C
import androidx.media3.common.MediaItem as ExoMediaItem
import androidx.media3.common.MimeTypes
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.TrackSelectionOverride
import androidx.media3.common.Tracks
import androidx.media3.common.VideoSize
import androidx.media3.datasource.okhttp.OkHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import kotlin.math.roundToLong
import kotlinx.coroutines.delay
import okhttp3.OkHttpClient

@SuppressLint("UnsafeOptInUsageError")
@Composable
fun PlayerScreen(
    viewModel: PlayerViewModel,
    client: OkHttpClient,
    isTelevision: Boolean,
    onBack: () -> Unit
) {
    val state by viewModel.state.collectAsState()
    val context = LocalContext.current
    val pictureInPicture = remember(context) {
        (context.applicationContext as CinecatApplication).container.pictureInPicture
    }
    val supportsPictureInPicture = remember(context) {
        context.packageManager.hasSystemFeature(PackageManager.FEATURE_PICTURE_IN_PICTURE)
    }
    val inPip = pictureInPicture.inPip
    val exoPlayer = remember(client, context) {
        ExoPlayer.Builder(context)
            .setMediaSourceFactory(
                DefaultMediaSourceFactory(OkHttpDataSource.Factory(client))
            )
            .setSeekBackIncrementMs(SEEK_INCREMENT_MS)
            .setSeekForwardIncrementMs(SEEK_INCREMENT_MS)
            .build()
    }
    var isPlaying by remember(exoPlayer) { mutableStateOf(false) }
    var tracks by remember(exoPlayer) { mutableStateOf<Tracks?>(null) }
    var showTracks by remember { mutableStateOf(false) }
    var playerView by remember { mutableStateOf<PlayerView?>(null) }
    val rootFocus = remember { FocusRequester() }

    fun saveCurrentProgress() {
        val duration = exoPlayer.duration
        if (duration > 0) {
            viewModel.saveProgress(
                position = exoPlayer.currentPosition / 1000.0,
                duration = duration / 1000.0
            )
        }
    }

    val pipEligible = isPipEligible(
        hasVideo = state is PlayerUiState.Ready,
        isPlaying = isPlaying,
        isTelevision = isTelevision,
        supportsPip = supportsPictureInPicture
    )
    LaunchedEffect(pipEligible) {
        pictureInPicture.eligible = pipEligible
    }
    LaunchedEffect(inPip) {
        if (inPip) {
            showTracks = false
        }
    }
    DisposableEffect(pictureInPicture, exoPlayer) {
        pictureInPicture.onStopRequested = {
            saveCurrentProgress()
            exoPlayer.pause()
        }
        onDispose {
            pictureInPicture.eligible = false
            pictureInPicture.aspectRatio = PipAspectRatio.Default
            pictureInPicture.onStopRequested = null
        }
    }

    DisposableEffect(exoPlayer) {
        val listener = object : Player.Listener {
            override fun onIsPlayingChanged(playing: Boolean) {
                if (!playing && isPlaying) {
                    saveCurrentProgress()
                }
                isPlaying = playing
            }

            override fun onTracksChanged(currentTracks: Tracks) {
                tracks = currentTracks
            }

            override fun onVideoSizeChanged(videoSize: VideoSize) {
                pictureInPicture.aspectRatio = pipAspectRatio(videoSize.width, videoSize.height)
            }

            override fun onPlayerError(error: PlaybackException) {
                viewModel.onPlaybackError(exoPlayer.currentPosition / 1000.0)
            }
        }
        exoPlayer.addListener(listener)
        onDispose {
            saveCurrentProgress()
            exoPlayer.removeListener(listener)
            exoPlayer.release()
        }
    }

    PlayerFullscreenEffect(isTelevision)
    BackHandler {
        saveCurrentProgress()
        onBack()
    }

    LaunchedEffect(exoPlayer, state) {
        when (val current = state) {
            is PlayerUiState.Ready -> {
                exoPlayer.setMediaItem(current.toMediaItem())
                exoPlayer.seekTo(
                    (current.initialPositionSeconds * 1000.0).roundToLong()
                )
                exoPlayer.prepare()
                exoPlayer.playWhenReady = true
            }
            is PlayerUiState.Error -> {
                exoPlayer.pause()
                exoPlayer.clearMediaItems()
            }
            PlayerUiState.Loading -> exoPlayer.pause()
        }
    }

    LaunchedEffect(exoPlayer, isPlaying) {
        while (isPlaying) {
            delay(10_000)
            if (exoPlayer.isPlaying) {
                saveCurrentProgress()
            }
        }
    }

    LaunchedEffect(state is PlayerUiState.Ready) {
        runCatching { rootFocus.requestFocus() }
    }

    // Compose retiene el foco y las teclas no llegan a la PlayerView: el mando se
    // gestiona aquí, antes que cualquier hijo.
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black)
            .focusRequester(rootFocus)
            .focusable()
            .onPreviewKeyEvent { event ->
                if (inPip || state !is PlayerUiState.Ready || showTracks) {
                    return@onPreviewKeyEvent false
                }
                val action: (() -> Unit) = when (event.key) {
                    Key.DirectionCenter, Key.Enter, Key.NumPadEnter, Key.Spacebar,
                    Key.MediaPlayPause -> {
                        { if (exoPlayer.isPlaying) exoPlayer.pause() else exoPlayer.play() }
                    }
                    Key.MediaPlay -> exoPlayer::play
                    Key.MediaPause -> exoPlayer::pause
                    Key.DirectionLeft, Key.MediaRewind -> exoPlayer::seekBack
                    Key.DirectionRight, Key.MediaFastForward -> exoPlayer::seekForward
                    Key.DirectionDown -> { {} }
                    Key.DirectionUp, Key.Menu -> { { showTracks = true } }
                    else -> return@onPreviewKeyEvent false
                }
                if (event.type == KeyEventType.KeyDown) {
                    action()
                    playerView?.showController()
                }
                true
            }
    ) {
        when (val current = state) {
            PlayerUiState.Loading -> if (!inPip) {
                Column(
                    modifier = Modifier.align(Alignment.Center),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    CircularProgressIndicator(color = MaterialTheme.colorScheme.primary)
                    Text(
                        text = stringResource(R.string.loading),
                        style = MaterialTheme.typography.bodyLarge,
                        color = Color.White
                    )
                }
            }
            is PlayerUiState.Error -> if (!inPip) {
                Column(
                    modifier = Modifier
                        .align(Alignment.Center)
                        .padding(32.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    Text(
                        text = current.message ?: stringResource(R.string.player_error),
                        style = MaterialTheme.typography.headlineSmall,
                        // El fondo es negro en ambos temas.
                        color = Color.White
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Button(onClick = viewModel::retry) {
                            Text(stringResource(R.string.retry))
                        }
                        OutlinedButton(onClick = onBack) {
                            Text(stringResource(R.string.back))
                        }
                    }
                }
            }
            is PlayerUiState.Ready -> {
                PlayerViewSurface(
                    exoPlayer = exoPlayer,
                    showControls = !inPip,
                    onCreated = { playerView = it }
                )
                if (!inPip) {
                    TextButton(
                        onClick = { showTracks = true },
                        modifier = Modifier
                            .align(Alignment.TopEnd)
                            .padding(16.dp)
                            .background(
                                color = Color.Black.copy(alpha = 0.55f),
                                shape = RoundedCornerShape(8.dp)
                            ),
                        colors = ButtonDefaults.textButtonColors(contentColor = Color.White)
                    ) {
                        Text(stringResource(R.string.player_tracks))
                    }
                    if (showTracks) {
                        TrackDialog(
                            player = exoPlayer,
                            tracks = tracks,
                            onDismiss = { showTracks = false }
                        )
                    }
                }
            }
        }
    }
}

@SuppressLint("UnsafeOptInUsageError")
@Composable
private fun PlayerViewSurface(
    exoPlayer: ExoPlayer,
    showControls: Boolean,
    onCreated: (PlayerView) -> Unit
) {
    AndroidView(
        factory = { context ->
            PlayerView(context).apply {
                player = exoPlayer
                useController = showControls
                keepScreenOn = true
                setShutterBackgroundColor(android.graphics.Color.BLACK)
                onCreated(this)
            }
        },
        update = { view ->
            view.player = exoPlayer
            view.keepScreenOn = true
            view.useController = showControls
            if (!showControls) {
                view.hideController()
            }
        },
        modifier = Modifier.fillMaxSize()
    )
}

@SuppressLint("UnsafeOptInUsageError")
@Composable
private fun TrackDialog(
    player: Player,
    tracks: Tracks?,
    onDismiss: () -> Unit
) {
    val groups = tracks?.groups.orEmpty()
        .filter { it.type == C.TRACK_TYPE_AUDIO || it.type == C.TRACK_TYPE_TEXT }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.player_tracks)) },
        confirmButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.back))
            }
        },
        text = {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                if (groups.isEmpty()) {
                    Text(stringResource(R.string.player_no_tracks))
                }
                groups.filter { it.type == C.TRACK_TYPE_AUDIO }.forEach { group ->
                    Text(
                        text = stringResource(R.string.player_audio),
                        style = MaterialTheme.typography.titleMedium
                    )
                    TrackGroupRows(player, group, C.TRACK_TYPE_AUDIO)
                }
                groups.filter { it.type == C.TRACK_TYPE_TEXT }.forEach { group ->
                    Text(
                        text = stringResource(R.string.player_subtitles),
                        style = MaterialTheme.typography.titleMedium
                    )
                    TrackGroupRows(player, group, C.TRACK_TYPE_TEXT)
                }
                if (groups.any { it.type == C.TRACK_TYPE_TEXT }) {
                    TextButton(
                        onClick = {
                            player.trackSelectionParameters =
                                player.trackSelectionParameters.buildUpon()
                                    .setTrackTypeDisabled(C.TRACK_TYPE_TEXT, true)
                                    .build()
                        }
                    ) {
                        Text(stringResource(R.string.player_disable_subtitles))
                    }
                }
            }
        }
    )
}

@SuppressLint("UnsafeOptInUsageError")
@Composable
private fun TrackGroupRows(player: Player, group: Tracks.Group, type: Int) {
    for (index in 0 until group.mediaTrackGroup.length) {
        val format = group.mediaTrackGroup.getFormat(index)
        val label = format.label?.takeIf { it.isNotBlank() }
            ?: format.language?.takeIf { it.isNotBlank() }
            ?: stringResource(R.string.player_tracks)
        val selected = group.isTrackSelected(index)
        TextButton(
            onClick = {
                val parameters = player.trackSelectionParameters.buildUpon()
                if (type == C.TRACK_TYPE_TEXT) {
                    parameters.setTrackTypeDisabled(C.TRACK_TYPE_TEXT, false)
                }
                parameters.setOverrideForType(
                    TrackSelectionOverride(group.mediaTrackGroup, index)
                )
                player.trackSelectionParameters = parameters.build()
            }
        ) {
            Text(
                if (selected) {
                    stringResource(R.string.player_track_selected, label)
                } else {
                    label
                }
            )
        }
    }
}

@Composable
private fun PlayerFullscreenEffect(isTelevision: Boolean) {
    val context = LocalContext.current
    val view = LocalView.current
    val activity = remember(context) { context.findActivity() }

    DisposableEffect(activity, view, isTelevision) {
        val window = activity?.window
        val controller = window?.let { WindowCompat.getInsetsController(it, view) }
        val previousOrientation = activity?.requestedOrientation
            ?: ActivityInfo.SCREEN_ORIENTATION_UNSPECIFIED
        controller?.systemBarsBehavior =
            WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
        controller?.hide(WindowInsetsCompat.Type.systemBars())
        if (!isTelevision) {
            activity?.requestedOrientation = ActivityInfo.SCREEN_ORIENTATION_SENSOR_LANDSCAPE
        }
        onDispose {
            controller?.show(WindowInsetsCompat.Type.systemBars())
            activity?.requestedOrientation = previousOrientation
        }
    }
}

@SuppressLint("UnsafeOptInUsageError")
private fun PlayerUiState.Ready.toMediaItem(): ExoMediaItem {
    val subtitleConfigurations = subtitles.map { subtitle ->
        ExoMediaItem.SubtitleConfiguration.Builder(Uri.parse(subtitle.url))
            .setMimeType(MimeTypes.TEXT_VTT)
            .setLanguage(subtitle.language)
            .setLabel(subtitle.label)
            .setSelectionFlags(
                if (subtitle.selected) C.SELECTION_FLAG_DEFAULT else 0
            )
            .build()
    }
    return ExoMediaItem.Builder()
        .setUri(streamUrl)
        .apply {
            if (isHls) {
                setMimeType(MimeTypes.APPLICATION_M3U8)
            }
        }
        .setSubtitleConfigurations(subtitleConfigurations)
        .build()
}

private const val SEEK_INCREMENT_MS = 10_000L

private tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}
