package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.focusable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.withFrameNanos
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
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.PhotoItem
import kotlinx.coroutines.delay

@Composable
fun PhotoViewerScreen(
    photos: List<PhotoItem>,
    initialIndex: Int,
    isTelevision: Boolean,
    onClose: () -> Unit
) {
    Dialog(
        onDismissRequest = onClose,
        properties = DialogProperties(
            usePlatformDefaultWidth = false,
            decorFitsSystemWindows = false
        )
    ) {
        CompositionLocalProvider(LocalIsTelevision provides isTelevision) {
            PhotoViewerContent(
                photos = photos,
                initialIndex = initialIndex,
                isTelevision = isTelevision,
                onClose = onClose
            )
        }
    }
}

@Composable
private fun PhotoViewerContent(
    photos: List<PhotoItem>,
    initialIndex: Int,
    isTelevision: Boolean,
    onClose: () -> Unit
) {
    var index by remember {
        mutableStateOf(initialIndex.coerceIn(0, (photos.size - 1).coerceAtLeast(0)))
    }
    var playing by remember { mutableStateOf(false) }
    var drag by remember { mutableStateOf(0f) }
    val rootFocus = remember { FocusRequester() }
    val threshold = with(LocalDensity.current) { 48.dp.toPx() }
    val current = photos.getOrNull(index)

    LaunchedEffect(isTelevision) {
        if (isTelevision) {
            withFrameNanos { }
            val focused = runCatching { rootFocus.requestFocus() }
            if (focused.isFailure) {
                withFrameNanos { }
                runCatching { rootFocus.requestFocus() }
            }
        }
    }

    LaunchedEffect(playing, index) {
        if (playing && photos.size > 1) {
            delay(5_000)
            if (playing) {
                index = nextPhotoIndex(index, photos.size)
            }
        }
    }

    fun stop() {
        playing = false
    }

    fun previous() {
        stop()
        index = previousPhotoIndex(index, photos.size)
    }

    fun next() {
        stop()
        index = nextPhotoIndex(index, photos.size)
    }

    fun togglePresentation() {
        if (photos.size > 1) {
            playing = !playing
        }
    }

    val swipeModifier = if (isTelevision) {
        Modifier
    } else {
        Modifier.pointerInput(photos.size) {
            detectHorizontalDragGestures(
                onDragStart = { drag = 0f },
                onHorizontalDrag = { _, amount -> drag += amount },
                onDragEnd = {
                    if (drag > threshold) {
                        previous()
                    } else if (drag < -threshold) {
                        next()
                    }
                    drag = 0f
                },
                onDragCancel = { drag = 0f }
            )
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black)
            .then(swipeModifier)
            .windowInsetsPadding(WindowInsets.safeDrawing)
            .focusRequester(rootFocus)
            .focusable()
            .onPreviewKeyEvent { event ->
                if (event.type != KeyEventType.KeyDown) {
                    return@onPreviewKeyEvent false
                }
                when (event.key) {
                    Key.DirectionLeft -> {
                        previous()
                        true
                    }
                    Key.DirectionRight -> {
                        next()
                        true
                    }
                    Key.DirectionCenter, Key.Enter, Key.NumPadEnter -> {
                        togglePresentation()
                        true
                    }
                    else -> false
                }
            }
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = stringResource(R.string.photos_viewer_counter, index + 1, photos.size),
                color = Color.White,
                style = MaterialTheme.typography.labelLarge
            )
            TextButton(
                onClick = onClose,
                modifier = Modifier.focusRing(RoundedCornerShape(8.dp))
            ) {
                Text(stringResource(R.string.photos_viewer_close), color = Color.White)
            }
        }

        if (current == null) {
            Text(
                text = stringResource(R.string.photos_empty),
                color = Color.White,
                modifier = Modifier.align(Alignment.CenterHorizontally)
            )
        } else {
            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth(),
                contentAlignment = Alignment.Center
            ) {
                AsyncImage(
                    model = photoOriginalPath(current.id),
                    contentDescription = stringResource(
                        R.string.photo_description,
                        current.name
                    ),
                    contentScale = ContentScale.Fit,
                    modifier = Modifier.fillMaxSize()
                )
            }
            Column(
                modifier = Modifier.fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(2.dp)
            ) {
                Text(
                    text = current.name,
                    color = Color.White,
                    style = MaterialTheme.typography.titleMedium
                )
                val date = photoDateLabel(current.takenAt)
                if (date.isNotEmpty()) {
                    Text(
                        text = date,
                        color = Color.White.copy(alpha = 0.72f),
                        style = MaterialTheme.typography.labelMedium
                    )
                }
            }
        }

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp, Alignment.CenterHorizontally),
            verticalAlignment = Alignment.CenterVertically
        ) {
            TextButton(
                onClick = ::previous,
                enabled = photos.size > 1,
                modifier = Modifier.focusRing(RoundedCornerShape(8.dp))
            ) {
                Text(stringResource(R.string.photos_previous), color = Color.White)
            }
            TextButton(
                onClick = ::togglePresentation,
                enabled = photos.size > 1,
                modifier = Modifier.focusRing(RoundedCornerShape(8.dp))
            ) {
                Text(
                    text = stringResource(
                        if (playing) R.string.photos_stop else R.string.photos_presentation
                    ),
                    color = Color.White
                )
            }
            TextButton(
                onClick = ::next,
                enabled = photos.size > 1,
                modifier = Modifier.focusRing(RoundedCornerShape(8.dp))
            ) {
                Text(stringResource(R.string.photos_next), color = Color.White)
            }
        }
    }
}
