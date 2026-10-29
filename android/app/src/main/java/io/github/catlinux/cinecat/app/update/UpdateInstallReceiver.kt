package io.github.catlinux.cinecat.app.update

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageInstaller
import android.os.Build
import io.github.catlinux.cinecat.app.CinecatApplication

class UpdateInstallReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (intent.getIntExtra(PackageInstaller.EXTRA_STATUS, -1)) {
            PackageInstaller.STATUS_PENDING_USER_ACTION -> {
                val userAction = intent.getParcelableExtraCompat(Intent.EXTRA_INTENT)
                if (userAction == null) {
                    reportFailure(context, "No se recibió la acción de instalación")
                    return
                }
                try {
                    context.startActivity(
                        userAction.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                    )
                } catch (error: Exception) {
                    reportFailure(context, error.message)
                }
            }
            PackageInstaller.STATUS_SUCCESS -> Unit
            else -> {
                val detail = intent.getStringExtra(PackageInstaller.EXTRA_STATUS_MESSAGE)
                reportFailure(context, detail)
            }
        }
    }

    private fun reportFailure(context: Context, detail: String?) {
        val application = context.applicationContext as? CinecatApplication
        application?.container?.appUpdater?.reportInstallFailure(detail)
    }

    @Suppress("DEPRECATION")
    private fun Intent.getParcelableExtraCompat(name: String): Intent? {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            getParcelableExtra(name, Intent::class.java)
        } else {
            getParcelableExtra(name)
        }
    }
}
