package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseDao
import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseEntity
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map

/** The exercises the user can pick from, kept on the device so they're there offline too. */
interface ExercisesRepository {
    /** The global and the user's own exercises, sorted by name, ignoring case. Empty until they've been fetched. */
    val exercises: Flow<List<Exercise>>

    /** Fetches the global exercises if none are stored yet, as on a fresh install. Needs no session. */
    suspend fun fetchGlobalIfNoneStored()

    /**
     * Replaces the stored global exercises, then the user's own, with the server's. Only the user's
     * own need a session. Stops at the first that fails, keeping what's stored.
     */
    suspend fun refresh(): RefreshResult
}

class RoomExercisesRepository(private val api: ExercisesApi, private val dao: ExerciseDao) : ExercisesRepository {

    override val exercises: Flow<List<Exercise>> = dao.observeExercises().map { entities ->
        entities
            .sortedWith(compareBy({ it.name.lowercase() }, { it.exerciseId.toString() }))
            .map { Exercise(it.exerciseId, it.name) }
    }

    override suspend fun fetchGlobalIfNoneStored() {
        if (dao.observeExercises().first().none { it.isGlobal }) replace(isGlobal = true, api.getGlobalExercises())
    }

    override suspend fun refresh(): RefreshResult {
        val global = replace(isGlobal = true, api.getGlobalExercises())
        if (global != RefreshResult.Success) return global
        return replace(isGlobal = false, api.getExercises())
    }

    private suspend fun replace(isGlobal: Boolean, result: ApiResult<List<Exercise>>): RefreshResult = when (result) {
        is ApiResult.Success -> {
            dao.replaceExercises(isGlobal, result.value.map { ExerciseEntity(it.id, it.name, isGlobal) })
            RefreshResult.Success
        }
        ApiResult.Unauthorized -> RefreshResult.NotLoggedIn
        ApiResult.NetworkError -> RefreshResult.NetworkError
        ApiResult.ServerError -> RefreshResult.ServerError
    }
}
