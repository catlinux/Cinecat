package io.github.catlinux.cinecat.app

import androidx.annotation.StringRes
import io.github.catlinux.cinecat.core.api.PlaySegment

enum class SegmentButtonKind {
    Intro,
    Credits
}

data class SegmentButton(
    @StringRes val labelRes: Int,
    val endSeconds: Double,
    val kind: SegmentButtonKind
)

// El inicio es inclusivo y el fin exclusivo: en el segundo exacto del final el
// botón ya no debe tapar la reproducción normal.
fun activeSegmentButton(
    positionSeconds: Double,
    segments: List<PlaySegment>
): SegmentButton? {
    if (!positionSeconds.isFinite()) {
        return null
    }
    val segment = segments.firstOrNull { candidate ->
        positionSeconds >= candidate.start && positionSeconds < candidate.end
    } ?: return null
    return when (segment.kind) {
        "intro" -> SegmentButton(
            labelRes = R.string.player_skip_intro,
            endSeconds = segment.end,
            kind = SegmentButtonKind.Intro
        )
        "credits" -> SegmentButton(
            labelRes = R.string.player_skip_credits,
            endSeconds = segment.end,
            kind = SegmentButtonKind.Credits
        )
        else -> null
    }
}
