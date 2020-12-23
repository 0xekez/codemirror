package com.chmod4.mirror.listeners

import com.chmod4.mirror.services.Session
import com.intellij.openapi.editor.event.SelectionEvent
import com.intellij.openapi.editor.event.SelectionListener

class MySelectionListener : SelectionListener {
    override fun selectionChanged(event: SelectionEvent) {
        super.selectionChanged(event)
        Session.updateSelection()
    }
}
