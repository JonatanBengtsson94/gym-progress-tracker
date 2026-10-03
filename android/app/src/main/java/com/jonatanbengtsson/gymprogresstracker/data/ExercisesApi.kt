package com.jonatanbengtsson.gymprogresstracker.data

import kotlin.uuid.Uuid

data class Exercise(val id: Uuid, val name: String)

interface ExercisesApi {
    /** Lists the user's own exercises together with the global ones, sorted by name. */
    suspend fun getExercises(): ApiResult<List<Exercise>>
}

class HttpExercisesApi(private val client: ApiClient) : ExercisesApi {

    override suspend fun getExercises(): ApiResult<List<Exercise>> = client.send("GET", "/exercises") { json ->
        json.getJSONArray("exercises").objects().map { exercise ->
            Exercise(id = Uuid.parse(exercise.getString("exercise_id")), name = exercise.getString("exercise_name"))
        }
    }
}
