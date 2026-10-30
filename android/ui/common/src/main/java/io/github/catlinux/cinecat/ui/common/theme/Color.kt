package io.github.catlinux.cinecat.ui.common.theme

import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.ui.graphics.Color

internal val Cream = Color(0xFFFBF7EE)
internal val CreamLight = Color(0xFFFFFDF8)
internal val Maroon = Color(0xFF5A1A1F)
internal val MaroonDark = Color(0xFF3A1013)
internal val Gold = Color(0xFFC99A2E)
internal val Orange = Color(0xFFA64618)
internal val Ink = Color(0xFF2B1719)
internal val TextOnDark = Color(0xFFFDF7EA)

internal val NightBackground = Color(0xFF171012)
internal val NightSurface = Color(0xFF26191C)
internal val NightPrimary = Color(0xFFF0C9C0)
internal val NightGold = Color(0xFFE0B54A)
internal val NightOrange = Color(0xFFF08A4B)
internal val NightText = Color(0xFFF4E6D2)

// Sin estos contenedores y superficies Material rellena con su paleta base
// (lila y rosa), que asoma en la pastilla del menú lateral y en los diálogos.
internal val GoldSoft = Color(0xFFF2E2B8)
internal val OrangeSoft = Color(0xFFF4D3BD)
internal val CreamDeep = Color(0xFFF3ECDD)
internal val Line = Color(0xFFD8CBB8)
internal val NightGoldSoft = Color(0xFF4A3A1C)
internal val NightOrangeSoft = Color(0xFF5C2A14)
internal val NightSurfaceHigh = Color(0xFF332226)
internal val NightLine = Color(0xFF5E4A4D)

internal val LightColors = lightColorScheme(
    primary = Maroon,
    onPrimary = TextOnDark,
    primaryContainer = MaroonDark,
    onPrimaryContainer = TextOnDark,
    secondary = Gold,
    onSecondary = Ink,
    secondaryContainer = GoldSoft,
    onSecondaryContainer = Maroon,
    tertiary = Orange,
    onTertiary = TextOnDark,
    tertiaryContainer = OrangeSoft,
    onTertiaryContainer = MaroonDark,
    background = Cream,
    onBackground = Ink,
    surface = CreamLight,
    onSurface = Ink,
    surfaceVariant = CreamLight,
    onSurfaceVariant = Ink,
    surfaceTint = Maroon,
    surfaceContainerLowest = CreamLight,
    surfaceContainerLow = CreamLight,
    surfaceContainer = Cream,
    surfaceContainerHigh = CreamDeep,
    surfaceContainerHighest = CreamDeep,
    outline = Line,
    outlineVariant = Line
)

internal val DarkColors = darkColorScheme(
    primary = NightPrimary,
    onPrimary = NightBackground,
    primaryContainer = MaroonDark,
    onPrimaryContainer = TextOnDark,
    secondary = NightGold,
    onSecondary = NightBackground,
    secondaryContainer = NightGoldSoft,
    onSecondaryContainer = NightText,
    tertiary = NightOrange,
    onTertiary = NightBackground,
    tertiaryContainer = NightOrangeSoft,
    onTertiaryContainer = NightText,
    background = NightBackground,
    onBackground = NightText,
    surface = NightSurface,
    onSurface = NightText,
    surfaceVariant = NightSurface,
    onSurfaceVariant = NightText,
    surfaceTint = NightPrimary,
    surfaceContainerLowest = NightBackground,
    surfaceContainerLow = NightSurface,
    surfaceContainer = NightSurface,
    surfaceContainerHigh = NightSurfaceHigh,
    surfaceContainerHighest = NightSurfaceHigh,
    outline = NightLine,
    outlineVariant = NightLine
)
