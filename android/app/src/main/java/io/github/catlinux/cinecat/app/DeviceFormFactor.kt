package io.github.catlinux.cinecat.app

import android.content.res.Configuration

fun isTelevision(uiModeType: Int): Boolean = uiModeType == Configuration.UI_MODE_TYPE_TELEVISION
