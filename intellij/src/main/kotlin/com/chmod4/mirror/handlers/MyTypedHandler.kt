package com.chmod4.mirror.handlers

import com.intellij.openapi.actionSystem.DataContext
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.editor.actionSystem.TypedActionHandler

class MyTypedHandler : TypedActionHandler {
    override fun execute(editor: Editor, c: Char, dataContext: DataContext) {
        val doc = editor.document;
        println(doc.text);
    }
}
