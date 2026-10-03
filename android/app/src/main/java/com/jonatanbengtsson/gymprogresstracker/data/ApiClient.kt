package com.jonatanbengtsson.gymprogresstracker.data

import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONException
import org.json.JSONObject
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import java.time.DateTimeException

sealed interface ApiResult<out T> {
    data class Success<T>(val value: T) : ApiResult<T>

    /** The server answered 401: the session has ended, or for a login, the credentials are wrong. */
    data object Unauthorized : ApiResult<Nothing>
    data object NetworkError : ApiResult<Nothing>
    data object ServerError : ApiResult<Nothing>
}

/** Sends requests to the backend, as the logged-in user unless said otherwise. */
class ApiClient(private val baseUrl: String, private val sessionRepository: SessionRepository) {

    /**
     * Sends a request as the logged-in user and hands a successful response's body to [parse]. A 401
     * ends the session the request was sent with, which logs the user out unless they've logged in
     * again since. Without a session nothing is sent.
     */
    suspend fun <T> send(method: String, path: String, body: JSONObject? = null, parse: (JSONObject) -> T): ApiResult<T> {
        val session = sessionRepository.session.value as? SessionState.LoggedIn ?: return ApiResult.Unauthorized
        val result = exchange(method, path, body, session.sessionId, parse)
        if (result == ApiResult.Unauthorized) sessionRepository.endSession(session.sessionId)
        return result
    }

    /** Sends a request without a session, which is how logging in works. */
    suspend fun <T> sendWithoutSession(method: String, path: String, body: JSONObject? = null, parse: (JSONObject) -> T): ApiResult<T> =
        exchange(method, path, body, sessionId = null, parse)

    private suspend fun <T> exchange(
        method: String,
        path: String,
        body: JSONObject?,
        sessionId: String?,
        parse: (JSONObject) -> T
    ): ApiResult<T> = withContext(Dispatchers.IO) {
        val connection = URL(baseUrl + path).openConnection() as HttpURLConnection
        try {
            connection.requestMethod = method
            connection.connectTimeout = TIMEOUT_MS
            connection.readTimeout = TIMEOUT_MS
            if (sessionId != null) connection.setRequestProperty("Authorization", "Bearer $sessionId")
            if (body != null) {
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "application/json")
                connection.outputStream.use { it.write(body.toString().toByteArray()) }
            }

            when (connection.responseCode) {
                in 200..299 -> {
                    val response = connection.inputStream.bufferedReader().use { it.readText() }
                    ApiResult.Success(parse(JSONObject(response)))
                }
                HttpURLConnection.HTTP_UNAUTHORIZED -> ApiResult.Unauthorized
                else -> ApiResult.ServerError
            }
        } catch (e: IOException) {
            Log.w(TAG, "$method $path to $baseUrl failed", e)
            ApiResult.NetworkError
        } catch (e: JSONException) {
            Log.w(TAG, "Unexpected response to $method $path", e)
            ApiResult.ServerError
        } catch (e: IllegalArgumentException) {
            Log.w(TAG, "Unexpected value in response to $method $path", e)
            ApiResult.ServerError
        } catch (e: DateTimeException) {
            Log.w(TAG, "Unexpected timestamp in response to $method $path", e)
            ApiResult.ServerError
        } finally {
            connection.disconnect()
        }
    }

    private companion object {
        const val TAG = "ApiClient"
        const val TIMEOUT_MS = 10_000
    }
}

/** The array's elements, each of which must be an object. */
fun JSONArray.objects(): List<JSONObject> = List(length()) { getJSONObject(it) }
