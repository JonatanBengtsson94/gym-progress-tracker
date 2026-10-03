package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseDao
import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseEntity
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

/** The exercises the user can pick from, kept on the device so they're there offline too. */
interface ExercisesRepository {
    /** The user's own and the global exercises, sorted by name. Empty until they've been fetched once. */
    val exercises: Flow<List<Exercise>>

    /** Replaces the stored exercises with the server's. They're kept as they are if that fails. */
    suspend fun refresh(sessionId: String): RefreshResult
}

class RoomExercisesRepository(private val api: ExercisesApi, private val dao: ExerciseDao) : ExercisesRepository {

    override val exercises: Flow<List<Exercise>> =
        dao.observeExercises().map { entities -> entities.map { Exercise(it.exerciseId, it.name) } }

    override suspend fun refresh(sessionId: String): RefreshResult = when (val result = api.getExercises(sessionId)) {
        is ExercisesResult.Success -> {
            dao.replaceExercises(result.exercises.mapIndexed { position, exercise -> ExerciseEntity(exercise.id, exercise.name, position) })
            RefreshResult.Success
        }
        ExercisesResult.SessionExpired -> RefreshResult.SessionExpired
        ExercisesResult.NetworkError -> RefreshResult.NetworkError
        ExercisesResult.ServerError -> RefreshResult.ServerError
    }
}
