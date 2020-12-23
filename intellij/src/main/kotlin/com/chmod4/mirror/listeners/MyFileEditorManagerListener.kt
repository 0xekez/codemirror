package com.chmod4.mirror.listeners

import com.chmod4.mirror.services.Session
import com.intellij.openapi.fileEditor.FileEditorManagerEvent
import com.intellij.openapi.fileEditor.FileEditorManagerListener

class MyFileEditorManagerListener : FileEditorManagerListener {
    // on selected open editor change
    override fun selectionChanged(event: FileEditorManagerEvent) {
        super.selectionChanged(event)
        Session.updateDataAndSelection()
    }
}
