package com.chmod4.mirror.services

import com.intellij.ide.BrowserUtil
import com.intellij.notification.NotificationDisplayType
import com.intellij.notification.NotificationGroup
import com.intellij.notification.NotificationType
import com.intellij.openapi.project.Project
import java.net.URI
import java.net.http.HttpClient
import java.net.http.WebSocket
import java.util.concurrent.CompletionStage
import kotlinx.serialization.*
import kotlinx.serialization.json.*

enum class MessageType(val value: String) {
    DATA("DATA"),
    URL("URL"),
    RESEND("RESEND"),
    SELECTION("SELECTION")
}

@Serializable
data class Message(val type: MessageType, val content: String)

object Session {
    private var ws: WebSocket? = null
    var wsURL: String? = null
    var project: Project? = null
    private val notifGroup = NotificationGroup("Session Notification Group", NotificationDisplayType.BALLOON, true)

    fun isConnected() = ws != null || wsURL != null

    fun create() {
        close()
        ws = HttpClient
            .newHttpClient()
            .newWebSocketBuilder()
            .buildAsync(URI.create("wss://mirror.chmod4.com/create"), WebSocketClient())
            .join()
    }

    fun close() {
        if (!isConnected()) {
            return
        }

        ws?.sendClose(WebSocket.NORMAL_CLOSURE, "")
        cleanupClose()
    }

    fun cleanupClose() {
        if (!isConnected()) {
            return
        }

        ws = null
        wsURL = null
        sendNotification("Mirroring session closed.", "", NotificationType.INFORMATION)
    }

    fun displayNoActiveSession() {
        sendNotification("No active mirroring session.", "", NotificationType.ERROR)
    }

    fun openWSURL() {
        if (isConnected()) {
            wsURL?.let { BrowserUtil.browse(it) }
        } else {
            displayNoActiveSession()
        }
    }

    fun updateDataAndSelection() {
        println("updateDataAndSelection")
    }

    private fun sendNotification(title: String, content: String, type: NotificationType) {
        notifGroup
            .createNotification(title, content, type)
            .notify(project)
    }

    class WebSocketClient : WebSocket.Listener {
        override fun onOpen(webSocket: WebSocket?) {
            super.onOpen(webSocket)
            sendNotification("Mirroring session started.", "", NotificationType.INFORMATION)
        }

        override fun onText(webSocket: WebSocket?, data: CharSequence?, last: Boolean): CompletionStage<*>? {
            val msg = Json.decodeFromString<Message>(data.toString())
            if (msg.type == MessageType.URL) {
                wsURL = msg.content
                openWSURL()
            } else if (msg.type == MessageType.RESEND) {
                updateDataAndSelection()
            }

            webSocket?.request(1)
            return null
        }

        override fun onClose(webSocket: WebSocket?, statusCode: Int, reason: String?): CompletionStage<*> {
            cleanupClose()
            return super.onClose(webSocket, statusCode, reason)
        }

        override fun onError(webSocket: WebSocket?, error: Throwable?) {
            println("error: " + error.toString())
            sendNotification("Error", error.toString(), NotificationType.ERROR)
            super.onError(webSocket, error)
        }
    }
}