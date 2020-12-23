package com.chmod4.mirror.services

import com.intellij.openapi.project.Project
import com.chmod4.mirror.MyBundle

class MyProjectService(project: Project) {

    init {
        println(MyBundle.message("projectService", project.name))
        Session.project = project
    }
}
