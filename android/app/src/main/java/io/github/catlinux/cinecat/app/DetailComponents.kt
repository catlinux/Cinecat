package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import io.github.catlinux.cinecat.core.api.CastCredit
import io.github.catlinux.cinecat.core.api.tmdbImage

@Composable
fun DetailBackButton(
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    IconButton(
        onClick = onBack,
        modifier = modifier
            .background(
                color = MaterialTheme.colorScheme.surface.copy(alpha = 0.78f),
                shape = CircleShape
            )
            .focusRing(CircleShape)
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
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    FilterChip(
        selected = active,
        onClick = onClick,
        modifier = modifier.focusRing(RoundedCornerShape(8.dp)),
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

/**
 * Cabecera de ficha para TV: cartel, datos, todas las acciones en una fila y
 * sinopsis, sobre el fondo. El foco baja siempre al primer elemento que el
 * llamador indique para que no salte a una fila lejana.
 */
@Composable
fun TvDetailHeader(
    title: String,
    backdropUrl: String?,
    posterUrl: String?,
    meta: String,
    overview: String?,
    primaryLabel: String?,
    initialFocusRequester: FocusRequester,
    nextFocusRequester: FocusRequester?,
    backFocusRequester: FocusRequester,
    favorite: Boolean,
    watchlist: Boolean,
    watched: Boolean,
    watchedLabel: String,
    onPrimary: (() -> Unit)?,
    onToggleFavorite: () -> Unit,
    onToggleWatchlist: () -> Unit,
    onToggleWatched: () -> Unit
) {
    val showPrimary = primaryLabel != null && onPrimary != null
    Box(modifier = Modifier.fillMaxWidth()) {
        DetailBackdrop(title = title, url = backdropUrl, height = 380.dp)
        // Velo crema opaco a la izquierda: los datos y la sinopsis van encima y,
        // sobre fondos claros, no se leían.
        val veil = MaterialTheme.colorScheme.background
        // matchParentSize: dentro de un LazyColumn fillMaxSize no tiene altura máxima.
        Box(
            modifier = Modifier
                .matchParentSize()
                .background(
                    Brush.horizontalGradient(
                        0f to veil.copy(alpha = 0.96f),
                        0.6f to veil.copy(alpha = 0.88f),
                        1f to veil.copy(alpha = 0.35f)
                    )
                )
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = 28.dp, end = 28.dp, top = 72.dp, bottom = 16.dp),
            horizontalArrangement = Arrangement.spacedBy(18.dp),
            verticalAlignment = Alignment.Top
        ) {
            DetailPoster(
                title = title,
                url = posterUrl,
                width = 180.dp
            )
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.headlineLarge,
                    color = MaterialTheme.colorScheme.primary,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
                if (meta.isNotBlank()) {
                    Text(
                        text = meta,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .horizontalScroll(rememberScrollState()),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    if (showPrimary) {
                        Button(
                            onClick = onPrimary,
                            modifier = Modifier
                                .focusRequester(initialFocusRequester)
                                .tvActionFocus(backFocusRequester, nextFocusRequester)
                                .focusRing(RoundedCornerShape(50))
                        ) {
                            Text(primaryLabel)
                        }
                    }
                    val firstActionModifier = if (!showPrimary) {
                        Modifier.focusRequester(initialFocusRequester)
                    } else {
                        Modifier
                    }
                    DetailActionChip(
                        label = stringResource(R.string.favorite),
                        active = favorite,
                        modifier = firstActionModifier.tvActionFocus(
                            backFocusRequester,
                            nextFocusRequester
                        ),
                        icon = {
                            Icon(
                                imageVector = if (favorite) {
                                    Icons.Filled.Favorite
                                } else {
                                    Icons.Filled.FavoriteBorder
                                },
                                contentDescription = null
                            )
                        },
                        onClick = onToggleFavorite
                    )
                    DetailActionChip(
                        label = stringResource(R.string.watchlist),
                        active = watchlist,
                        modifier = Modifier.tvActionFocus(
                            backFocusRequester,
                            nextFocusRequester
                        ),
                        icon = {
                            Icon(
                                painter = painterResource(R.drawable.ic_watchlist),
                                contentDescription = null
                            )
                        },
                        onClick = onToggleWatchlist
                    )
                    DetailActionChip(
                        label = watchedLabel,
                        active = watched,
                        modifier = Modifier.tvActionFocus(
                            backFocusRequester,
                            nextFocusRequester
                        ),
                        icon = {
                            Icon(
                                imageVector = Icons.Filled.CheckCircle,
                                contentDescription = null
                            )
                        },
                        onClick = onToggleWatched
                    )
                }
                Text(
                    text = overview?.trim()
                        .takeIf { !it.isNullOrEmpty() }
                        ?: stringResource(R.string.no_synopsis),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 4,
                    overflow = TextOverflow.Ellipsis
                )
            }
        }
    }
}

@Composable
private fun Modifier.tvActionFocus(
    back: FocusRequester,
    next: FocusRequester?
): Modifier {
    if (!LocalIsTelevision.current) {
        return this
    }
    return focusProperties {
        up = back
        if (next != null) {
            down = next
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

@Composable
fun DetailBackdrop(
    title: String,
    url: String?,
    height: Dp
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(height)
            .background(MaterialTheme.colorScheme.surface)
    ) {
        if (url != null) {
            AsyncImage(
                model = url,
                contentDescription = stringResource(R.string.backdrop_description, title),
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize()
            )
        }
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    Brush.verticalGradient(
                        colors = listOf(
                            Color.Transparent,
                            Color.Transparent,
                            MaterialTheme.colorScheme.background
                        )
                    )
                )
        )
    }
}

@Composable
fun DetailSynopsis(
    overview: String?,
    horizontalPadding: Dp
) {
    Column(
        modifier = Modifier.padding(start = horizontalPadding, end = horizontalPadding, top = 20.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        DetailSectionTitle(stringResource(R.string.synopsis))
        val text = overview?.trim().orEmpty()
        if (text.isEmpty()) {
            Text(
                text = stringResource(R.string.no_synopsis),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        } else {
            var expanded by rememberSaveable(text) { mutableStateOf(false) }
            var hasOverflow by remember(text) { mutableStateOf(false) }
            Text(
                text = text,
                style = MaterialTheme.typography.bodyLarge,
                maxLines = if (expanded) Int.MAX_VALUE else 6,
                overflow = TextOverflow.Ellipsis,
                onTextLayout = { result ->
                    if (!expanded) {
                        hasOverflow = result.hasVisualOverflow
                    }
                }
            )
            if (hasOverflow || expanded) {
                TextButton(
                    onClick = { expanded = !expanded },
                    modifier = Modifier.focusRing(RoundedCornerShape(50))
                ) {
                    Text(
                        stringResource(
                            if (expanded) R.string.read_less else R.string.read_more
                        )
                    )
                }
            }
        }
    }
}

@Composable
fun DetailCastRow(
    cast: List<CastCredit>,
    horizontalPadding: Dp
) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = horizontalPadding),
        horizontalArrangement = Arrangement.spacedBy(14.dp)
    ) {
        items(cast) { credit ->
            DetailCastMember(credit)
        }
    }
}

@Composable
private fun DetailCastMember(credit: CastCredit) {
    Column(
        modifier = Modifier.width(104.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        val photoUrl = tmdbImage(credit.photoUrl.takeIf { it.isNotBlank() }, "w185")
        Box(
            modifier = Modifier
                .size(64.dp)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.surfaceVariant),
            contentAlignment = Alignment.Center
        ) {
            if (photoUrl != null) {
                AsyncImage(
                    model = photoUrl,
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize()
                )
            } else {
                Text(
                    text = castInitials(credit.name),
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.primary
                )
            }
        }
        Text(
            text = credit.name,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis
        )
        if (credit.character.isNotBlank()) {
            Text(
                text = credit.character,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis
            )
        }
    }
}

private fun castInitials(name: String): String {
    return name.split(' ')
        .filter { it.isNotBlank() }
        .take(2)
        .joinToString("") { part -> part.first().uppercase() }
}
