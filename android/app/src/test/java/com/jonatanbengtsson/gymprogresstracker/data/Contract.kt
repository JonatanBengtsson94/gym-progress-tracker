package com.jonatanbengtsson.gymprogresstracker.data

/** The API's example request or response named [name], which the backend's tests check against too. */
fun contract(name: String): String =
    checkNotNull(object {}.javaClass.classLoader?.getResource(name)) { "No contract file $name" }.readText()
