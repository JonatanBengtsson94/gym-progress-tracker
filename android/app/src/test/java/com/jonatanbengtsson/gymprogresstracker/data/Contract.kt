package com.jonatanbengtsson.gymprogresstracker.data

import org.json.JSONArray
import org.json.JSONObject

/** The API's example request or response named [name], which the backend's tests check against too. */
fun contract(name: String): String =
    checkNotNull(object {}.javaClass.classLoader?.getResource(name)) { "No contract file $name" }.readText()

/** The JSON as maps and lists, which compare by content regardless of key order. */
fun JSONObject.toKotlin(): Any = toKotlinValue()

private fun Any.toKotlinValue(): Any = when (this) {
    is JSONObject -> keys().asSequence().associateWith { get(it).toKotlinValue() }
    is JSONArray -> List(length()) { get(it).toKotlinValue() }
    else -> this
}
