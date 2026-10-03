package com.jonatanbengtsson.gymprogresstracker.data

import android.util.Log
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONException
import org.json.JSONObject
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import kotlin.uuid.Uuid

data class Exercise(val id: Uuid, val name: String)

sealed interface ExercisesResult {
    data class Success(val exercises: List<Exercise>) : ExercisesResult
    data object SessionExpired : ExercisesResult
    data object NetworkError : ExercisesResult
    data object ServerError : ExercisesResult
}

interface ExercisesApi {
    /** Lists the user's own exercises together with the global ones, sorted by name. */
    suspend fun getExercises(sessionId: String): ExercisesResult
}

class HttpExercisesApi(private val baseUrl: String) : ExercisesApi {

    override suspend fun getExercises(sessionId: String): ExercisesResult = withContext(Dispatchers.IO) {
        val connection = URL("$baseUrl/exercises").openConnection() as HttpURLConnection
        try {
            connection.requestMethod = "GET"
            connection.connectTimeout = TIMEOUT_MS
            connection.readTimeout = TIMEOUT_MS
            connection.setRequestProperty("Authorization", "Bearer $sessionId")

            when (connection.responseCode) {
                HttpURLConnection.HTTP_OK -> {
                    val response = connection.inputStream.bufferedReader().use { it.readText() }
                    ExercisesResult.Success(parseExercises(JSONObject(response)))
                }
                HttpURLConnection.HTTP_UNAUTHORIZED -> ExercisesResult.SessionExpired
                else -> ExercisesResult.ServerError
            }
        } catch (e: IOException) {
            Log.w(TAG, "Exercises request to $baseUrl failed", e)
            ExercisesResult.NetworkError
        } catch (e: JSONException) {
            Log.w(TAG, "Unexpected exercises response", e)
            ExercisesResult.ServerError
        } catch (e: IllegalArgumentException) {
            Log.w(TAG, "Unexpected exercise id in exercises response", e)
            ExercisesResult.ServerError
        } finally {
            connection.disconnect()
        }
    }

    private fun parseExercises(json: JSONObject): List<Exercise> {
        val exercises = json.getJSONArray("exercises")
        return List(exercises.length()) {
            val exercise = exercises.getJSONObject(it)
            Exercise(id = Uuid.parse(exercise.getString("exercise_id")), name = exercise.getString("exercise_name"))
        }
    }

    private companion object {
        const val TAG = "HttpExercisesApi"
        const val TIMEOUT_MS = 10_000
    }
}
