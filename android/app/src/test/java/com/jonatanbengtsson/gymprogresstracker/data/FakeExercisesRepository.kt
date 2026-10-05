package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.flow.MutableStateFlow

/** Keeps the exercises in memory. Each refresh suspends until the test completes [refreshResult]. Counts the fetches of the global exercises. */
class FakeExercisesRepository(exercises: List<Exercise> = emptyList()) : ExercisesRepository {

    override val exercises = MutableStateFlow(exercises)
    var refreshResult = CompletableDeferred<RefreshResult>()
    var refreshes = 0
    var globalFetches = 0

    override suspend fun fetchGlobalIfNoneStored() {
        globalFetches++
    }

    override suspend fun refresh(): RefreshResult {
        refreshes++
        return refreshResult.await()
    }
}
