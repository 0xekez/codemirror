package com.chmod4.mirror.services

import com.chmod4.mirror.listeners.MyCaretListener
import com.chmod4.mirror.listeners.MyDocumentListener
import com.chmod4.mirror.listeners.MyFileEditorManagerListener
import com.chmod4.mirror.listeners.MySelectionListener
import com.intellij.ide.BrowserUtil
import com.intellij.notification.NotificationDisplayType
import com.intellij.notification.NotificationGroup
import com.intellij.notification.NotificationType
import com.intellij.openapi.Disposable
import com.intellij.openapi.components.service
import com.intellij.openapi.editor.EditorFactory
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.fileEditor.FileEditorManagerListener
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.Disposer
import java.net.URI
import java.net.http.HttpClient
import java.net.http.WebSocket
import java.util.concurrent.CompletionStage
import kotlinx.serialization.*
import kotlinx.serialization.json.*
import java.util.Timer
import java.util.concurrent.CompletableFuture
import kotlin.concurrent.schedule

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
    var disposable: Disposable? = null

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

    private fun sendMessage(type: MessageType, content: String) {
        if (!isConnected()) {
            return
        }

        val msg = Json.encodeToString(Message(type, content))
        CompletableFuture.runAsync { ws?.sendText(msg, true)?.get() }
    }

    fun updateSelection() {
        if (project == null) {
            return
        }

        val editor = FileEditorManager.getInstance(project!!).selectedTextEditor ?: return

        val start = editor.selectionModel.selectionStart
        var length = editor.selectionModel.selectionEnd - start
        val selection = String.format("%d %d", start, length)

        sendMessage(MessageType.SELECTION, selection)
    }

    fun updateDataAndSelection() {
        if (project == null) {
            return
        }

        val editor = FileEditorManager.getInstance(project!!).selectedTextEditor ?: return
        val text = editor.document.text
        sendMessage(MessageType.DATA, text)
        updateSelection()
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
        if (disposable != null && !Disposer.isDisposed(disposable!!)) {
            Disposer.dispose(disposable!!)
            disposable = null
        }
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

    private fun sendNotification(title: String, content: String, type: NotificationType) {
        notifGroup
            .createNotification(title, content, type)
            .notify(project)
    }

    class WebSocketClient : WebSocket.Listener {
        override fun onOpen(webSocket: WebSocket?) {
            super.onOpen(webSocket)

            // opened, start listening for stuff

            disposable = Disposer.newDisposable()
            val projectService = project?.service<MyProjectService>()!!
            Disposer.register(projectService, disposable!!)

            val multicaster = EditorFactory.getInstance().eventMulticaster
            multicaster.addDocumentListener(MyDocumentListener(), disposable!!)
            multicaster.addSelectionListener(MySelectionListener(), disposable!!)
            multicaster.addCaretListener(MyCaretListener(), disposable!!)
            // on selected open editor change
            project?.messageBus
                ?.connect(disposable!!)
                ?.subscribe(FileEditorManagerListener.FILE_EDITOR_MANAGER, MyFileEditorManagerListener())

            sendNotification("Mirroring session started.", "", NotificationType.INFORMATION)
        }

        override fun onText(webSocket: WebSocket?, data: CharSequence?, last: Boolean): CompletionStage<*>? {
            val msg = Json.decodeFromString<Message>(data.toString())
            if (msg.type == MessageType.URL) {
                wsURL = msg.content
                openWSURL()
            } else if (msg.type == MessageType.RESEND) {
//                updateDataAndSelection()
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