package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.MediaItem
import io.github.catlinux.cinecat.core.api.PlaybackProgress
import io.github.catlinux.cinecat.core.api.Series
import io.github.catlinux.cinecat.core.api.tmdbImage

@Composable
fun MediaPosterCard(
    item: MediaItem,
    width: Dp,
    onClick: () -> Unit
) {
    PosterCard(
        title = item.title,
        year = item.year,
        posterUrl = tmdbImage(item.posterUrl, "w342"),
        progress = item.progress,
        width = width,
        onClick = onClick
    )
}

@Composable
fun SeriesPosterCard(
    series: Series,
    width: Dp,
    onClick: () -> Unit
) {
    PosterCard(
        title = series.title,
        year = series.year,
        posterUrl = tmdbImage(series.posterUrl, "w342"),
        progress = null,
        width = width,
        onClick = onClick
    )
}

@Composable
fun EpisodeCard(
    item: MediaItem,
    width: Dp,
    onClick: () -> Unit
) {
    val imageUrl = tmdbImage(
        item.stillUrl ?: item.seriesBackdropUrl ?: item.backdropUrl,
        "w780"
    )
    val seriesTitle = item.seriesTitle ?: item.title
    Column(
        modifier = Modifier
            .width(width)
            .clickable(onClick = onClick)
            .tvFocusPadding()
    ) {
        Artwork(
            url = imageUrl,
            title = item.title,
            description = stringResource(R.string.episode_image_description, item.title),
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(16f / 9f)
        ) {
            EpisodeProgress(item.progress)
        }
        Spacer(Modifier.height(8.dp))
        Text(
            text = seriesTitle,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            style = MaterialTheme.typography.titleLarge.copy(
                fontSize = 16.sp,
                lineHeight = 20.sp,
                fontWeight = FontWeight.SemiBold
            ),
            color = MaterialTheme.colorScheme.onBackground
        )
        val code = formatEpisodeCode(item.season, item.episode)
        if (code.isNotEmpty()) {
            Text(
                text = code,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.secondary
            )
        }
        Text(
            text = item.episodeTitle ?: item.title,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}

@Composable
private fun PosterCard(
    title: String,
    year: Int?,
    posterUrl: String?,
    progress: PlaybackProgress?,
    width: Dp,
    onClick: () -> Unit
) {
    Column(
        modifier = Modifier
            .width(width)
            .clickable(onClick = onClick)
            .tvFocusPadding()
    ) {
        Artwork(
            url = posterUrl,
            title = title,
            description = stringResource(R.string.poster_description, title),
            modifier = Modifier
                .fillMaxWidth()
                .aspectRatio(2f / 3f)
        ) {
            MediaProgress(progress)
        }
        Spacer(Modifier.height(8.dp))
        Text(
            text = title,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            style = MaterialTheme.typography.titleLarge.copy(
                fontSize = 15.sp,
                lineHeight = 19.sp,
                fontWeight = FontWeight.SemiBold
            ),
            color = MaterialTheme.colorScheme.onBackground
        )
        year?.let {
            Text(
                text = it.toString(),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun Artwork(
    url: String?,
    title: String,
    description: String,
    modifier: Modifier,
    overlay: @Composable () -> Unit = {}
) {
    Box(
        modifier = modifier
            .clip(RoundedCornerShape(4.dp))
            .background(MaterialTheme.colorScheme.surface)
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .fillMaxHeight()
                .padding(12.dp),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 4,
                overflow = TextOverflow.Ellipsis
            )
        }
        if (url != null) {
            AsyncImage(
                model = url,
                contentDescription = description,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxWidth().fillMaxHeight()
            )
        }
        Box(
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
        ) {
            overlay()
        }
    }
}

@Composable
private fun MediaProgress(progress: PlaybackProgress?) {
    val fraction = progressFraction(progress)
    if (fraction > 0f) {
        ProgressBar(fraction)
    }
}

@Composable
private fun EpisodeProgress(progress: PlaybackProgress?) {
    val fraction = progressFraction(progress)
    if (fraction > 0f) {
        ProgressBar(fraction)
    }
}

@Composable
private fun ProgressBar(fraction: Float) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(4.dp)
            .background(Color.Black.copy(alpha = 0.45f))
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth(fraction)
                .fillMaxHeight()
                .background(MaterialTheme.colorScheme.primary)
        )
    }
}

private fun progressFraction(progress: PlaybackProgress?): Float {
    if (progress == null || progress.durationSeconds <= 0.0) {
        return 0f
    }
    return (progress.positionSeconds / progress.durationSeconds)
        .coerceIn(0.0, 1.0)
        .toFloat()
}
