package com.chmod4.mirror.actions

import com.chmod4.mirror.services.Session
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent

class CreateSessionAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        Session.create()
    }
}
