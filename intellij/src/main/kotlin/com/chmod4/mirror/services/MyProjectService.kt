package com.chmod4.mirror.services

import com.chmod4.mirror.MyBundle
import com.intellij.openapi.Disposable
import com.intellij.openapi.project.Project

class MyProjectService(project: Project) : Disposable {
    init {
        println(MyBundle.message("projectService", project.name))
        Session.init(project)
    }

    override fun dispose() {
        println("Project service disposed")
        Session.close()
    }
}
