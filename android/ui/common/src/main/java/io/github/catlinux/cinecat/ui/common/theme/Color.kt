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

internal val LightColors = lightColorScheme(
    primary = Maroon,
    onPrimary = TextOnDark,
    primaryContainer = MaroonDark,
    onPrimaryContainer = TextOnDark,
    secondary = Gold,
    onSecondary = Ink,
    tertiary = Orange,
    onTertiary = TextOnDark,
    background = Cream,
    onBackground = Ink,
    surface = CreamLight,
    onSurface = Ink,
    surfaceVariant = CreamLight,
    onSurfaceVariant = Ink
)

internal val DarkColors = darkColorScheme(
    primary = NightPrimary,
    onPrimary = NightBackground,
    primaryContainer = MaroonDark,
    onPrimaryContainer = TextOnDark,
    secondary = NightGold,
    onSecondary = NightBackground,
    tertiary = NightOrange,
    onTertiary = NightBackground,
    background = NightBackground,
    onBackground = NightText,
    surface = NightSurface,
    onSurface = NightText,
    surfaceVariant = NightSurface,
    onSurfaceVariant = NightText
)
