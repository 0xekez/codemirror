package com.chmod4.mirror.services

import com.intellij.openapi.project.Project
import com.chmod4.mirror.MyBundle
import com.intellij.openapi.Disposable

class MyProjectService(project: Project) : Disposable {
    init {
        println(MyBundle.message("projectService", project.name))
        Session.project = project
    }

    override fun dispose() {
        println("Project service disposed")
    }
}
