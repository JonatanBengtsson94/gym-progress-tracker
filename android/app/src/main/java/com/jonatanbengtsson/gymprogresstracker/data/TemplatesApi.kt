package com.jonatanbengtsson.gymprogresstracker.data

import org.json.JSONObject
import java.time.Instant
import java.time.OffsetDateTime
import kotlin.uuid.Uuid

data class WorkoutSet(val reps: Int, val weightGrams: Int)

data class WorkoutExercise(val exerciseId: Uuid, val name: String, val sets: List<WorkoutSet>)

data class LatestWorkout(
    val workoutId: Uuid,
    val startedAt: Instant,
    val completedAt: Instant,
    val exercises: List<WorkoutExercise>
)

/** [latestWorkout] is null when no workout has been logged under the template yet. */
data class WorkoutTemplate(val id: Uuid, val name: String, val latestWorkout: LatestWorkout?)

interface TemplatesApi {
    /** Lists the user's templates, most recently performed first. */
    suspend fun getTemplates(): ApiResult<List<WorkoutTemplate>>
}

class HttpTemplatesApi(private val client: ApiClient) : TemplatesApi {

    override suspend fun getTemplates(): ApiResult<List<WorkoutTemplate>> = client.send("GET", "/templates") { json ->
        json.getJSONArray("templates").objects().map { template ->
            WorkoutTemplate(
                id = Uuid.parse(template.getString("template_id")),
                name = template.getString("template_name"),
                latestWorkout = template.optJSONObject("latest_workout")?.let(::parseLatestWorkout)
            )
        }
    }

    private fun parseLatestWorkout(json: JSONObject) = LatestWorkout(
        workoutId = Uuid.parse(json.getString("workout_id")),
        startedAt = parseTimestamp(json.getString("started_at")),
        completedAt = parseTimestamp(json.getString("completed_at")),
        exercises = json.getJSONArray("exercises").objects().map { exercise ->
            WorkoutExercise(
                exerciseId = Uuid.parse(exercise.getString("exercise_id")),
                name = exercise.getString("exercise_name"),
                sets = exercise.getJSONArray("sets").objects().map { set ->
                    WorkoutSet(reps = set.getInt("reps"), weightGrams = set.getInt("weight_grams"))
                }
            )
        }
    )

    // The backend sends RFC 3339 timestamps, which may carry a UTC offset other than Z.
    private fun parseTimestamp(value: String): Instant = OffsetDateTime.parse(value).toInstant()
}
