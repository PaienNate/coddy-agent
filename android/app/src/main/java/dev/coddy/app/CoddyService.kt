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
 * the process. Every line the server prints, and the exit code when it ends,
 * goes to [CoddyState], which the launcher draws and keeps in a file.
 */
class CoddyService : Service() {

    companion object {
        private const val TAG = "CoddyService"
        private const val CHANNEL_ID = "coddy"
        private const val NOTIFICATION_ID = 1

        const val ACTION_STOP = "dev.coddy.app.STOP"
    }

    @Volatile
    private var process: Process? = null
    private val stopping = AtomicBoolean(false)

    override fun onCreate() {
        super.onCreate()
        createChannel()
        startForeground(NOTIFICATION_ID, notification())
        CoddyState.setStopped()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopServer()
            stopSelf()
            return START_NOT_STICKY
        }
        if (process == null) startServer()
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onDestroy() {
        stopServer()
        super.onDestroy()
    }

    private fun startServer() {
        val runtime = CoddyRuntime(this)
        if (!runtime.isReady()) {
            CoddyState.log(this, "libcoddy.so is missing from the APK's native libraries")
            CoddyState.log(this, "nativeLibraryDir=${applicationInfo.nativeLibraryDir}")
            notifyText("Coddy could not start: the binary is missing")
            return
        }
        stopping.set(false)
        CoddyState.setStarting()
        Thread({
            try {
                // First run writes the config and links the busybox applets;
                // that is file work and one exec, so it stays off the main
                // thread together with the server itself.
                val builder = runtime.prepare()
                logVersion(runtime)
                val started = builder.start()
                process = started
                CoddyState.setRunning(started.pid())
                Log.i(TAG, "coddy serve started, pid=${started.pid()}")
                // Both streams go to the launcher and the file, so a crash is
                // readable on the phone instead of only over adb.
                started.inputStream.bufferedReader().forEachLine { line ->
                    Log.i(TAG, line)
                    CoddyState.log(this, line)
                }
                val code = started.waitFor()
                CoddyState.log(this, "coddy serve exited with code $code")
                process = null
                CoddyState.setExited(code)
                if (!stopping.get()) {
                    notifyText("Coddy stopped unexpectedly (exit $code)")
                }
            } catch (e: Exception) {
                Log.e(TAG, "coddy serve failed to start", e)
                CoddyState.log(this, "failed to start: ${e}")
                process = null
                CoddyState.setExited(-1)
                notifyText("Coddy could not start: ${e.message}")
            }
        }, "coddy-serve").start()
    }

    /** The binary's own build line, before the server says anything. */
    private fun logVersion(runtime: CoddyRuntime) {
        try {
            val p = ProcessBuilder(runtime.binary.absolutePath, "-v")
                .redirectErrorStream(true)
                .start()
            val line = p.inputStream.bufferedReader().readText().trim()
            p.waitFor()
            if (line.isNotEmpty()) CoddyState.log(this, line)
        } catch (e: Exception) {
            CoddyState.log(this, "version check failed: $e")
        }
    }

    private fun stopServer() {
        stopping.set(true)
        process?.let { p ->
            CoddyState.log(this, "stopping coddy serve (pid ${p.pid()})")
            p.destroy()
            try {
                p.waitFor()
            } catch (_: InterruptedException) {
                p.destroyForcibly()
            }
        }
        process = null
        CoddyState.setStopped()
    }

    private fun notifyText(text: String) {
        val manager = getSystemService(NotificationManager::class.java) ?: return
        manager.notify(
            NOTIFICATION_ID,
            Notification.Builder(this, CHANNEL_ID)
                .setContentTitle(getString(R.string.app_name))
                .setContentText(text)
                .setSmallIcon(android.R.drawable.stat_notify_error)
                .setContentIntent(openAppIntent())
                .build(),
        )
    }

    private fun notification(): Notification =
        Notification.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.app_name))
            .setContentText(getString(R.string.notification_running))
            .setSmallIcon(android.R.drawable.stat_sys_download_done)
            .setOngoing(true)
            .setContentIntent(openAppIntent())
            .build()

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
