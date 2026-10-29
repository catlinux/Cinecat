package io.github.catlinux.cinecat.app

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusDirection
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.unit.dp
import androidx.tv.material3.Border
import androidx.tv.material3.Button as TvButton
import androidx.tv.material3.ButtonDefaults as TvButtonDefaults

/**
 * Un campo de texto de una línea consume arriba y abajo para mover el cursor, y con el
 * mando de la TV no hay otra forma de salir de él: aquí esas flechas mueven el foco.
 * En el móvil no llegan flechas salvo con un teclado físico, donde también conviene.
 */
@Composable
fun Modifier.dpadFieldNavigation(): Modifier {
    val focusManager = LocalFocusManager.current
    return onPreviewKeyEvent { event ->
        val direction = when (event.key) {
            Key.DirectionUp -> FocusDirection.Up
            Key.DirectionDown -> FocusDirection.Down
            else -> return@onPreviewKeyEvent false
        }
        if (event.type == KeyEventType.KeyDown) focusManager.moveFocus(direction)
        true
    }
}

@Composable
fun PrimaryActionButton(
    isTelevision: Boolean,
    label: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    onClick: () -> Unit
) {
    if (isTelevision) {
        val shape = RoundedCornerShape(10.dp)
        TvButton(
            onClick = onClick,
            modifier = modifier,
            enabled = enabled,
            shape = TvButtonDefaults.shape(shape = shape),
            colors = TvButtonDefaults.colors(
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary,
                focusedContainerColor = MaterialTheme.colorScheme.primary,
                focusedContentColor = MaterialTheme.colorScheme.onPrimary
            ),
            border = TvButtonDefaults.border(
                border = Border.None,
                focusedBorder = Border(
                    border = BorderStroke(4.dp, MaterialTheme.colorScheme.tertiary),
                    shape = shape
                )
            ),
            contentPadding = PaddingValues(horizontal = 36.dp, vertical = 18.dp)
        ) {
            // El Text de Material 3 no lee el color de contenido del botón de TV.
            Text(text = label, style = MaterialTheme.typography.titleLarge, color = MaterialTheme.colorScheme.onPrimary)
        }
    } else {
        Button(
            onClick = onClick,
            modifier = modifier
                .fillMaxWidth()
                .heightIn(min = 52.dp),
            enabled = enabled
        ) {
            Text(text = label, style = MaterialTheme.typography.titleMedium)
        }
    }
}

@Composable
fun SecondaryActionButton(
    isTelevision: Boolean,
    label: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    onClick: () -> Unit
) {
    if (isTelevision) {
        val shape = RoundedCornerShape(10.dp)
        TvButton(
            onClick = onClick,
            modifier = modifier,
            enabled = enabled,
            shape = TvButtonDefaults.shape(shape = shape),
            colors = TvButtonDefaults.colors(
                containerColor = MaterialTheme.colorScheme.surfaceVariant,
                contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
                focusedContainerColor = MaterialTheme.colorScheme.surfaceVariant,
                focusedContentColor = MaterialTheme.colorScheme.onSurfaceVariant
            ),
            border = TvButtonDefaults.border(
                border = Border.None,
                focusedBorder = Border(
                    border = BorderStroke(4.dp, MaterialTheme.colorScheme.tertiary),
                    shape = shape
                )
            ),
            contentPadding = PaddingValues(horizontal = 32.dp, vertical = 16.dp)
        ) {
            Text(text = label, style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    } else {
        OutlinedButton(
            onClick = onClick,
            modifier = modifier
                .fillMaxWidth()
                .heightIn(min = 48.dp),
            enabled = enabled
        ) {
            Text(text = label, style = MaterialTheme.typography.titleMedium)
        }
    }
}
