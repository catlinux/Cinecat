package io.github.catlinux.cinecat.ui.common.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import io.github.catlinux.cinecat.ui.common.R

internal val Fraunces = FontFamily(
    Font(R.font.fraunces_variable, FontWeight.Normal),
    Font(R.font.fraunces_variable, FontWeight.Medium),
    Font(R.font.fraunces_variable, FontWeight.SemiBold),
    Font(R.font.fraunces_variable, FontWeight.Bold)
)

internal val MarquesinaTypography = Typography(
    displayLarge = TextStyle(
        fontFamily = Fraunces,
        fontWeight = FontWeight.Bold,
        fontSize = 64.sp,
        lineHeight = 70.sp,
        letterSpacing = (-1).sp
    ),
    displayMedium = TextStyle(
        fontFamily = Fraunces,
        fontWeight = FontWeight.Bold,
        fontSize = 52.sp,
        lineHeight = 60.sp,
        letterSpacing = (-0.5).sp
    ),
    displaySmall = TextStyle(
        fontFamily = Fraunces,
        fontWeight = FontWeight.SemiBold,
        fontSize = 40.sp,
        lineHeight = 46.sp
    ),
    headlineLarge = TextStyle(
        fontFamily = Fraunces,
        fontWeight = FontWeight.SemiBold,
        fontSize = 40.sp,
        lineHeight = 46.sp
    ),
    headlineMedium = TextStyle(
        fontFamily = Fraunces,
        fontWeight = FontWeight.SemiBold,
        fontSize = 32.sp,
        lineHeight = 38.sp
    ),
    headlineSmall = TextStyle(
        fontFamily = Fraunces,
        fontWeight = FontWeight.Medium,
        fontSize = 26.sp,
        lineHeight = 34.sp
    ),
    titleLarge = TextStyle(
        fontFamily = Fraunces,
        fontWeight = FontWeight.SemiBold,
        fontSize = 24.sp,
        lineHeight = 30.sp
    )
)
