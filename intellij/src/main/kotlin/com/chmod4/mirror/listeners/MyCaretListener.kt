package com.chmod4.mirror.listeners

import com.chmod4.mirror.services.Session
import com.intellij.openapi.editor.event.CaretEvent
import com.intellij.openapi.editor.event.CaretListener

class MyCaretListener : CaretListener {
    override fun caretPositionChanged(event: CaretEvent) {
        super.caretPositionChanged(event)
        Session.updateSelection()
    }
}
