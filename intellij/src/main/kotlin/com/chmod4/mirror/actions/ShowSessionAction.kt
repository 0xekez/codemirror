package com.chmod4.mirror.actions

import com.chmod4.mirror.services.Session
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent

class ShowSessionAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        if (Session.isConnected()) {
            Session.openWSURL()
        } else {
            Session.displayNoActiveSession()
        }
    }
}
