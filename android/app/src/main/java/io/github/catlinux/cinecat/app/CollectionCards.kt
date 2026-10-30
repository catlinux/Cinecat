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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.Collection
import io.github.catlinux.cinecat.core.api.tmdbImage

@Composable
fun CollectionPosterCard(
    collection: Collection,
    width: Dp,
    onClick: () -> Unit,
    compact: Boolean = false
) {
    val posterUrl = tmdbImage(collection.posterUrl, "w342")
    val titleLineHeight = 15.sp
    val titleHeight = with(LocalDensity.current) { titleLineHeight.toDp() * 3f }
    Column(
        modifier = Modifier
            .width(width)
            .clickable(onClick = onClick)
            .tvFocusPadding()
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
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
                    text = collection.name,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                    maxLines = 4,
                    overflow = TextOverflow.Ellipsis
                )
            }
            if (posterUrl != null) {
                AsyncImage(
                    model = posterUrl,
                    contentDescription = stringResource(R.string.poster_description, collection.name),
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxWidth().fillMaxHeight()
                )
            }
        }
        Spacer(Modifier.height(8.dp))
        Text(
            text = collection.name,
            modifier = if (compact) Modifier.heightIn(min = titleHeight) else Modifier,
            maxLines = if (compact) 3 else 2,
            overflow = TextOverflow.Ellipsis,
            style = MaterialTheme.typography.titleLarge.copy(
                fontSize = if (compact) 12.sp else 15.sp,
                lineHeight = if (compact) titleLineHeight else 19.sp,
                fontWeight = FontWeight.SemiBold
            ),
            color = MaterialTheme.colorScheme.onBackground
        )
        Text(
            text = pluralStringResource(
                R.plurals.collection_item_count,
                collection.itemCount.toInt(),
                collection.itemCount
            ),
            style = if (compact) {
                MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp, lineHeight = 12.sp)
            } else {
                MaterialTheme.typography.labelMedium
            },
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}
