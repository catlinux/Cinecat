package io.github.catlinux.cinecat.app

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.IndicationNodeFactory
import androidx.compose.foundation.interaction.FocusInteraction
import androidx.compose.foundation.interaction.InteractionSource
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.foundation.layout.padding
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.drawscope.ContentDrawScope
import androidx.compose.ui.graphics.drawOutline
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.node.DelegatableNode
import androidx.compose.ui.node.DrawModifierNode
import androidx.compose.ui.node.invalidateDraw
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch

val LocalIsTelevision = staticCompositionLocalOf { false }

/**
 * Indica el foco de las tarjetas con un borde, sin depender
 * del ripple de Material 3, que en la TV apenas se distingue.
 */
class FocusIndication(private val color: Color) : IndicationNodeFactory {
    override fun create(interactionSource: InteractionSource): DelegatableNode =
        FocusIndicationNode(color, interactionSource)

    override fun equals(other: Any?): Boolean =
        other is FocusIndication && other.color == color

    override fun hashCode(): Int = color.hashCode()
}

private class FocusIndicationNode(
    private val color: Color,
    private val interactionSource: InteractionSource
) : Modifier.Node(), DrawModifierNode {
    private val progress = Animatable(0f)

    override fun onAttach() {
        coroutineScope.launch {
            interactionSource.interactions.collect { interaction ->
                when (interaction) {
                    is FocusInteraction.Focus -> progress.animateTo(1f, tween(150))
                    is FocusInteraction.Unfocus -> progress.animateTo(0f, tween(150))
                }
            }
        }
        coroutineScope.launch {
            snapshotFlow { progress.value }.collect { invalidateDraw() }
        }
    }

    override fun ContentDrawScope.draw() {
        // Sin zoom: la lista recorta la tarjeta ampliada en los bordes de la fila.
        val value = progress.value
        drawContent()
        if (value <= 0f) {
            return
        }
        val stroke = 3.dp.toPx()
        drawRoundRect(
            color = color.copy(alpha = value),
            topLeft = Offset(stroke / 2f, stroke / 2f),
            size = Size(
                (size.width - stroke).coerceAtLeast(0f),
                (size.height - stroke).coerceAtLeast(0f)
            ),
            cornerRadius = CornerRadius(10.dp.toPx()),
            style = Stroke(stroke)
        )
    }
}

@Composable
fun Modifier.focusRing(shape: Shape = RoundedCornerShape(10.dp)): Modifier {
    if (!LocalIsTelevision.current) {
        return this
    }
    val color = MaterialTheme.colorScheme.tertiary
    var focused by remember { mutableStateOf(false) }
    return this
        .onFocusChanged { focused = it.hasFocus }
        .drawWithContent {
            drawContent()
            if (focused) {
                val stroke = 3.dp.toPx()
                val inset = stroke / 2f
                translate(inset, inset) {
                    drawOutline(
                        outline = shape.createOutline(
                            size = Size(
                                (size.width - stroke).coerceAtLeast(0f),
                                (size.height - stroke).coerceAtLeast(0f)
                            ),
                            layoutDirection = layoutDirection,
                            density = this
                        ),
                        color = color,
                        style = Stroke(width = stroke)
                    )
                }
            }
        }
}

// Margen para que el marco de foco no pise el título ni el cartel de la tarjeta.
@Composable
fun Modifier.tvFocusPadding(): Modifier =
    if (LocalIsTelevision.current) this.padding(6.dp) else this

