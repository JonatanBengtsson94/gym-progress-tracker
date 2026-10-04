package com.jonatanbengtsson.gymprogresstracker.data

import org.json.JSONArray
import org.json.JSONObject
import java.time.Instant
import kotlin.uuid.Uuid

/** A workout as it was logged, with [name] being the name of the template it's logged under. */
data class FinishedWorkout(
    val name: String,
    val startedAt: Instant,
    val completedAt: Instant,
    val exercises: List<WorkoutExercise>
)

interface WorkoutsApi {
    /**
     * Makes the user's workout [workoutId] match [workout], creating it under the template [templateId]
     * if it doesn't exist yet, so sending the same workout again changes nothing.
     */
    suspend fun putWorkout(workoutId: Uuid, templateId: Uuid, workout: FinishedWorkout): ApiResult<Unit>
}

class HttpWorkoutsApi(private val client: ApiClient) : WorkoutsApi {

    override suspend fun putWorkout(workoutId: Uuid, templateId: Uuid, workout: FinishedWorkout): ApiResult<Unit> {
        val body = JSONObject()
            .put("template_id", templateId.toString())
            .put("started_at", workout.startedAt.toString())
            .put("completed_at", workout.completedAt.toString())
            .put("exercises", JSONArray(workout.exercises.map { exercise ->
                JSONObject()
                    .put("exercise_id", exercise.exerciseId.toString())
                    .put("sets", JSONArray(exercise.sets.map { JSONObject().put("reps", it.reps).put("weight_grams", it.weightGrams) }))
            }))
        return client.send("PUT", "/workouts/$workoutId", body) {}
    }
}
