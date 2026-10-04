package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.flow.MutableStateFlow
import kotlin.uuid.Uuid

/** Records each save, which suspends until the test completes [saveDone]. Each sync suspends until the test completes [syncResult]. */
class FakeWorkoutsRepository : WorkoutsRepository {

    override val pendingCount = MutableStateFlow(0)
    var saveDone = CompletableDeferred<Unit>()
    val saves = mutableListOf<Pair<Uuid, FinishedWorkout>>()
    var syncResult = CompletableDeferred<SyncResult>()
    var syncs = 0

    override suspend fun save(workoutId: Uuid, workout: FinishedWorkout) {
        saves += workoutId to workout
        saveDone.await()
    }

    override suspend fun sync(): SyncResult {
        syncs++
        return syncResult.await()
    }
}
