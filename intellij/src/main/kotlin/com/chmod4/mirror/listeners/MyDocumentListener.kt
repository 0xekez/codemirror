package com.chmod4.mirror.listeners

import com.chmod4.mirror.services.Session
import com.intellij.openapi.editor.event.DocumentEvent
import com.intellij.openapi.editor.event.DocumentListener

class MyDocumentListener : DocumentListener {
    override fun documentChanged(event: DocumentEvent) {
        super.documentChanged(event)
        Session.updateDataAndSelection()
    }
}
