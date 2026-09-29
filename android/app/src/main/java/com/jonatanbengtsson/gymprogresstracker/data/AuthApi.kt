package com.jonatanbengtsson.gymprogresstracker.data

import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONException
import org.json.JSONObject
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL

sealed interface LoginResult {
    data class Success(val sessionId: String) : LoginResult
    data object InvalidCredentials : LoginResult
    data object NetworkError : LoginResult
    data object ServerError : LoginResult
}

class AuthApi(private val baseUrl: String) {

    suspend fun login(username: String, password: String): LoginResult = withContext(Dispatchers.IO) {
        val body = JSONObject()
            .put("username", username)
            .put("password", password)
            .toString()

        val connection = URL("$baseUrl/login").openConnection() as HttpURLConnection
        try {
            connection.requestMethod = "POST"
            connection.connectTimeout = TIMEOUT_MS
            connection.readTimeout = TIMEOUT_MS
            connection.doOutput = true
            connection.setRequestProperty("Content-Type", "application/json")
            connection.outputStream.use { it.write(body.toByteArray()) }

            when (connection.responseCode) {
                HttpURLConnection.HTTP_OK -> {
                    val response = connection.inputStream.bufferedReader().use { it.readText() }
                    LoginResult.Success(JSONObject(response).getString("session_id"))
                }
                HttpURLConnection.HTTP_UNAUTHORIZED -> LoginResult.InvalidCredentials
                else -> LoginResult.ServerError
            }
        } catch (e: IOException) {
            Log.w(TAG, "Login request to $baseUrl failed", e)
            LoginResult.NetworkError
        } catch (e: JSONException) {
            Log.w(TAG, "Unexpected login response", e)
            LoginResult.ServerError
        } finally {
            connection.disconnect()
        }
    }

    private companion object {
        const val TAG = "AuthApi"
        const val TIMEOUT_MS = 10_000
    }
}
