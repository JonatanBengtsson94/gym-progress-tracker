package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.flow.MutableStateFlow

/** Keeps the exercises in memory. Each refresh suspends until the test completes [refreshResult]. */
class FakeExercisesRepository(exercises: List<Exercise> = emptyList()) : ExercisesRepository {

    override val exercises = MutableStateFlow(exercises)
    var refreshResult = CompletableDeferred<RefreshResult>()
    val refreshes = mutableListOf<String>()

    override suspend fun refresh(sessionId: String): RefreshResult {
        refreshes += sessionId
        return refreshResult.await()
    }
}
