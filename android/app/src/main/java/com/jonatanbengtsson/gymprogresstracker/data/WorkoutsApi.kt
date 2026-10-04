package com.jonatanbengtsson.gymprogresstracker.data

import org.json.JSONArray
import org.json.JSONObject
import java.time.Instant
import kotlin.uuid.Uuid

data class FinishedExercise(val exerciseId: Uuid, val sets: List<WorkoutSet>)

/** A workout to log under the template [templateId], or under a new template named [templateName] when that's null. */
data class FinishedWorkout(
    val templateId: Uuid?,
    val templateName: String,
    val startedAt: Instant,
    val completedAt: Instant,
    val exercises: List<FinishedExercise>
)

interface WorkoutsApi {
    /**
     * Makes the user's workout [workoutId] match [workout], creating it if it doesn't exist yet, so
     * sending the same workout again changes nothing.
     */
    suspend fun putWorkout(workoutId: Uuid, workout: FinishedWorkout): ApiResult<Unit>
}

class HttpWorkoutsApi(private val client: ApiClient) : WorkoutsApi {

    override suspend fun putWorkout(workoutId: Uuid, workout: FinishedWorkout): ApiResult<Unit> =
        client.send("PUT", "/workouts/$workoutId", workout.toJson()) {}

    private fun FinishedWorkout.toJson() = JSONObject().apply {
        if (templateId != null) put("template_id", templateId.toString()) else put("template_name", templateName)
        put("started_at", startedAt.toString())
        put("completed_at", completedAt.toString())
        put("exercises", JSONArray(exercises.map { exercise ->
            JSONObject()
                .put("exercise_id", exercise.exerciseId.toString())
                .put("sets", JSONArray(exercise.sets.map { JSONObject().put("reps", it.reps).put("weight_grams", it.weightGrams) }))
        }))
    }
}
