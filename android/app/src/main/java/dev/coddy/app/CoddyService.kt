package dev.coddy.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.util.Log
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Runs `coddy serve` for the life of the app.
 *
 * It is a foreground service because the process must survive the activity
 * being backgrounded: a turn, a background task or a long tool call keeps
 * going while the person is elsewhere, and Android otherwise freezes or kills
 * the process. The notification is the price of that, and it is the one place
 * the phone says Coddy is working.
 */
class CoddyService : Service() {

    companion object {
        private const val TAG = "CoddyService"
        private const val CHANNEL_ID = "coddy"
        private const val NOTIFICATION_ID = 1

        @Volatile
        var running: Boolean = false
            private set
    }

    private var process: Process? = null
    private val stopped = AtomicBoolean(false)

    override fun onCreate() {
        super.onCreate()
        createChannel()
        startForeground(NOTIFICATION_ID, notification())
        startServer()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (!running && process == null) {
            startServer()
        }
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onDestroy() {
        stopped.set(true)
        process?.destroy()
        process = null
        running = false
        super.onDestroy()
    }

    private fun startServer() {
        val runtime = CoddyRuntime(this)
        if (!runtime.isReady()) {
            Log.e(TAG, "libcoddy.so is missing from the APK's native libraries")
            notifyText("Coddy could not start: the binary is missing")
            return
        }
        if (running) return

        Thread({
            try {
                val builder = runtime.prepare()
                Log.i(TAG, "starting ${builder.command().joinToString(" ")}")
                val started = builder.start()
                process = started
                running = true
                // Drain the server's output into logcat (adb logcat -s Coddy),
                // so a failed start is diagnosable from a phone.
                started.inputStream.bufferedReader().forEachLine { line ->
                    Log.i(TAG, line)
                }
                val code = started.waitFor()
                Log.w(TAG, "coddy serve exited with $code")
                running = false
                process = null
                if (!stopped.get()) {
                    notifyText("Coddy stopped unexpectedly (exit $code)")
                }
            } catch (e: Exception) {
                Log.e(TAG, "coddy serve failed to start", e)
                notifyText("Coddy could not start: ${e.message}")
            }
        }, "coddy-serve").start()
    }

    private fun notifyText(text: String) {
        val manager = getSystemService(NotificationManager::class.java)
        manager?.notify(
            NOTIFICATION_ID,
            Notification.Builder(this, CHANNEL_ID)
                .setContentTitle(getString(R.string.app_name))
                .setContentText(text)
                .setSmallIcon(android.R.drawable.stat_notify_error)
                .setContentIntent(openAppIntent())
                .build(),
        )
    }

    private fun notification(): Notification {
        val builder = Notification.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.app_name))
            .setContentText(getString(R.string.notification_running))
            .setSmallIcon(android.R.drawable.stat_sys_download_done)
            .setOngoing(true)
            .setContentIntent(openAppIntent())
        return builder.build()
    }

    private fun openAppIntent(): PendingIntent {
        val intent = Intent(this, MainActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        return PendingIntent.getActivity(
            this,
            0,
            intent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = getSystemService(NotificationManager::class.java) ?: return
        if (manager.getNotificationChannel(CHANNEL_ID) != null) return
        val channel = NotificationChannel(
            CHANNEL_ID,
            getString(R.string.notification_channel),
            NotificationManager.IMPORTANCE_LOW,
        ).apply {
            description = getString(R.string.notification_channel_description)
        }
        manager.createNotificationChannel(channel)
    }
}
