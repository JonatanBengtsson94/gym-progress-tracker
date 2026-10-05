package com.jonatanbengtsson.gymprogresstracker.data

import org.json.JSONObject
import kotlin.uuid.Uuid

data class Exercise(val id: Uuid, val name: String)

interface ExercisesApi {
    /** Lists the exercises every user can see, sorted by name. Needs no session. */
    suspend fun getGlobalExercises(): ApiResult<List<Exercise>>

    /** Lists the user's own exercises, without the global ones, sorted by name. */
    suspend fun getExercises(): ApiResult<List<Exercise>>
}

class HttpExercisesApi(private val client: ApiClient) : ExercisesApi {

    override suspend fun getGlobalExercises(): ApiResult<List<Exercise>> =
        client.sendWithoutSession("GET", "/global-exercises", parse = ::parseExercises)

    override suspend fun getExercises(): ApiResult<List<Exercise>> = client.send("GET", "/exercises", parse = ::parseExercises)

    private fun parseExercises(json: JSONObject): List<Exercise> =
        json.getJSONArray("exercises").objects().map { exercise ->
            Exercise(id = Uuid.parse(exercise.getString("exercise_id")), name = exercise.getString("exercise_name"))
        }
}
