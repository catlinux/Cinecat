package io.github.catlinux.cinecat.app

import android.content.res.Configuration
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import io.github.catlinux.cinecat.ui.common.theme.CinecatTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val isTv = isTelevision(resources.configuration.uiMode and Configuration.UI_MODE_TYPE_MASK)
        setContent {
            CinecatTheme {
                if (isTv) {
                    TvHomeScreen()
                } else {
                    MobileHomeScreen()
                }
            }
        }
    }
}
