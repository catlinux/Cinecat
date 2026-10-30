package io.github.catlinux.cinecat.app

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.LazyGridState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import io.github.catlinux.cinecat.core.api.LetterIndex
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull

/**
 * Devuelve la inicial de la fila que contiene [index], usando los tramos que
 * publica el servidor. Fuera de esos tramos (por ejemplo, el pie de carga) no
 * hay letra que resaltar.
 */
internal fun letterForIndex(letters: List<LetterIndex>, index: Int): String? {
    if (index < 0) {
        return null
    }
    return letters.firstOrNull { letter ->
        index >= letter.offset && index < letter.offset + letter.count
    }?.letter
}

/**
 * Índice vertical de iniciales para la TV. Solo se dibuja en televisión y cuando
 * el listado viene ordenado por título; en el resto de casos no ocupa espacio.
 */
@Composable
fun TvLetterIndex(
    letters: List<LetterIndex>,
    gridState: LazyGridState,
    ensureLoaded: suspend (Int) -> Unit,
    modifier: Modifier = Modifier
) {
    if (!LocalIsTelevision.current || letters.isEmpty()) {
        return
    }

    val scope = rememberCoroutineScope()
    val firstVisibleIndex by remember(gridState) {
        derivedStateOf { gridState.firstVisibleItemIndex }
    }
    val currentLetter = letterForIndex(letters, firstVisibleIndex)

    Column(
        modifier = modifier
            .width(38.dp)
            .fillMaxHeight()
            .padding(horizontal = 2.dp, vertical = 12.dp)
    ) {
        letters.forEach { letter ->
            LetterButton(
                letter = letter,
                current = letter.letter == currentLetter,
                onClick = {
                    scope.launch {
                        ensureLoaded(letter.offset)
                        // scrollToItem se limita a los elementos que la cuadrícula
                        // ya conoce: hay que esperar a que reciba las páginas nuevas.
                        withTimeoutOrNull(2_000) {
                            snapshotFlow { gridState.layoutInfo.totalItemsCount }
                                .first { it > letter.offset }
                        }
                        gridState.scrollToItem(letter.offset)
                        // Tras cargar varias páginas la cuadrícula aún se asienta y
                        // desplaza la primera colocación unas filas.
                        delay(300)
                        gridState.scrollToItem(letter.offset)
                    }
                },
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
            )
        }
    }
}

@Composable
private fun LetterButton(
    letter: LetterIndex,
    current: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val shape = RoundedCornerShape(7.dp)
    Box(
        modifier = modifier
            .clip(shape)
            .background(
                if (current) {
                    MaterialTheme.colorScheme.primary.copy(alpha = 0.14f)
                } else {
                    Color.Transparent
                }
            )
            .focusRing(shape)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = letter.letter,
            style = MaterialTheme.typography.titleLarge.copy(
                fontSize = 13.sp,
                lineHeight = 15.sp
            ),
            color = if (current) {
                MaterialTheme.colorScheme.primary
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            }
        )
    }
}
