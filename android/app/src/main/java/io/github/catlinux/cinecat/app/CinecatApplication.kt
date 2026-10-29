package io.github.catlinux.cinecat.app

import android.app.Application

class CinecatApplication : Application() {
    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
    }
}
