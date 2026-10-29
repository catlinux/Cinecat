package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage

@Composable
fun DetailBackButton(
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    IconButton(
        onClick = onBack,
        modifier = modifier.background(
            color = MaterialTheme.colorScheme.surface.copy(alpha = 0.78f),
            shape = CircleShape
        )
    ) {
        Icon(
            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
            contentDescription = stringResource(R.string.back),
            tint = MaterialTheme.colorScheme.primary
        )
    }
}

@Composable
fun DetailActionChip(
    label: String,
    active: Boolean,
    icon: @Composable () -> Unit,
    onClick: () -> Unit
) {
    FilterChip(
        selected = active,
        onClick = onClick,
        label = { Text(label) },
        leadingIcon = icon,
        colors = FilterChipDefaults.filterChipColors(
            selectedContainerColor = MaterialTheme.colorScheme.primary,
            selectedLabelColor = MaterialTheme.colorScheme.onPrimary,
            selectedLeadingIconColor = MaterialTheme.colorScheme.onPrimary
        )
    )
}

@Composable
fun DetailPoster(
    title: String,
    url: String?,
    width: Dp
) {
    Box(
        modifier = Modifier
            .width(width)
            .aspectRatio(2f / 3f)
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
                contentDescription = stringResource(R.string.poster_description, title),
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxWidth().fillMaxHeight()
            )
        }
    }
}

@Composable
fun DetailSectionTitle(
    title: String,
    modifier: Modifier = Modifier
) {
    Text(
        text = title,
        modifier = modifier,
        style = MaterialTheme.typography.headlineSmall,
        color = MaterialTheme.colorScheme.primary
    )
}

@Composable
fun DetailMetaText(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium),
        color = MaterialTheme.colorScheme.onSurfaceVariant
    )
}

@Composable
fun DetailBodyText(
    text: String,
    maxLines: Int = Int.MAX_VALUE
) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodyLarge,
        color = MaterialTheme.colorScheme.onBackground,
        maxLines = maxLines,
        overflow = if (maxLines == Int.MAX_VALUE) TextOverflow.Clip else TextOverflow.Ellipsis
    )
}
