package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CompletableDeferred
import kotlin.uuid.Uuid

/** Records each save, which suspends until the test completes [saveResult]. */
class FakeWorkoutsRepository : WorkoutsRepository {

    var saveResult = CompletableDeferred<ApiResult<Unit>>()
    val saves = mutableListOf<Pair<Uuid, FinishedWorkout>>()

    override suspend fun save(workoutId: Uuid, workout: FinishedWorkout): ApiResult<Unit> {
        saves += workoutId to workout
        return saveResult.await()
    }
}
