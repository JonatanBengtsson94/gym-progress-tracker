package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.flow.MutableStateFlow
import kotlin.uuid.Uuid

/** Records each save, which suspends until the test completes [saveDone]. Each send suspends until the test completes [sendResult]. */
class FakeWorkoutsRepository : WorkoutsRepository {

    override val pendingCount = MutableStateFlow(0)
    var saveDone = CompletableDeferred<Unit>()
    val saves = mutableListOf<Pair<Uuid, FinishedWorkout>>()
    var sendResult = CompletableDeferred<SyncResult>()
    var sends = 0

    override suspend fun save(workoutId: Uuid, workout: FinishedWorkout) {
        saves += workoutId to workout
        saveDone.await()
    }

    override suspend fun send(): SyncResult {
        sends++
        return sendResult.await()
    }
}
