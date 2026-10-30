package io.github.catlinux.cinecat.app.update

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import io.github.catlinux.cinecat.app.MainActivity

class UpdateRelaunchReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_MY_PACKAGE_REPLACED) {
            return
        }
        runCatching {
            context.startActivity(
                Intent(context, MainActivity::class.java)
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            )
        }
    }
}
