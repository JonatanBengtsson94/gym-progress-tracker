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
import java.time.Instant
import java.time.OffsetDateTime
import java.time.format.DateTimeParseException

data class WorkoutSet(val reps: Int, val weightGrams: Int)

data class WorkoutExercise(val exerciseId: Long, val name: String, val sets: List<WorkoutSet>)

data class LatestWorkout(
    val workoutId: Long,
    val startedAt: Instant,
    val completedAt: Instant,
    val exercises: List<WorkoutExercise>
)

/** [latestWorkout] is null when no workout has been logged under the template yet. */
data class WorkoutTemplate(val id: Long, val name: String, val latestWorkout: LatestWorkout?)

sealed interface TemplatesResult {
    data class Success(val templates: List<WorkoutTemplate>) : TemplatesResult
    data object SessionExpired : TemplatesResult
    data object NetworkError : TemplatesResult
    data object ServerError : TemplatesResult
}

interface TemplatesApi {
    /** Lists the user's templates, most recently performed first. */
    suspend fun getTemplates(sessionId: String): TemplatesResult
}

class HttpTemplatesApi(private val baseUrl: String) : TemplatesApi {

    override suspend fun getTemplates(sessionId: String): TemplatesResult = withContext(Dispatchers.IO) {
        val connection = URL("$baseUrl/templates").openConnection() as HttpURLConnection
        try {
            connection.requestMethod = "GET"
            connection.connectTimeout = TIMEOUT_MS
            connection.readTimeout = TIMEOUT_MS
            connection.setRequestProperty("Authorization", "Bearer $sessionId")

            when (connection.responseCode) {
                HttpURLConnection.HTTP_OK -> {
                    val response = connection.inputStream.bufferedReader().use { it.readText() }
                    TemplatesResult.Success(parseTemplates(JSONObject(response)))
                }
                HttpURLConnection.HTTP_UNAUTHORIZED -> TemplatesResult.SessionExpired
                else -> TemplatesResult.ServerError
            }
        } catch (e: IOException) {
            Log.w(TAG, "Templates request to $baseUrl failed", e)
            TemplatesResult.NetworkError
        } catch (e: JSONException) {
            Log.w(TAG, "Unexpected templates response", e)
            TemplatesResult.ServerError
        } catch (e: DateTimeParseException) {
            Log.w(TAG, "Unexpected timestamp in templates response", e)
            TemplatesResult.ServerError
        } finally {
            connection.disconnect()
        }
    }

    private fun parseTemplates(json: JSONObject): List<WorkoutTemplate> =
        json.getJSONArray("templates").objects().map { template ->
            WorkoutTemplate(
                id = template.getLong("template_id"),
                name = template.getString("template_name"),
                latestWorkout = template.optJSONObject("latest_workout")?.let(::parseLatestWorkout)
            )
        }

    private fun parseLatestWorkout(json: JSONObject) = LatestWorkout(
        workoutId = json.getLong("workout_id"),
        startedAt = parseTimestamp(json.getString("started_at")),
        completedAt = parseTimestamp(json.getString("completed_at")),
        exercises = json.getJSONArray("exercises").objects().map { exercise ->
            WorkoutExercise(
                exerciseId = exercise.getLong("exercise_id"),
                name = exercise.getString("exercise_name"),
                sets = exercise.getJSONArray("sets").objects().map { set ->
                    WorkoutSet(reps = set.getInt("reps"), weightGrams = set.getInt("weight_grams"))
                }
            )
        }
    )

    // The backend sends RFC 3339 timestamps, which may carry a UTC offset other than Z.
    private fun parseTimestamp(value: String): Instant = OffsetDateTime.parse(value).toInstant()

    private fun JSONArray.objects(): List<JSONObject> = List(length()) { getJSONObject(it) }

    private companion object {
        const val TAG = "HttpTemplatesApi"
        const val TIMEOUT_MS = 10_000
    }
}
