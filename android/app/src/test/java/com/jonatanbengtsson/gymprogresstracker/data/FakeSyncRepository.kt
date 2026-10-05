package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CompletableDeferred

/** Each sync suspends until the test completes [syncResult]. */
class FakeSyncRepository : SyncRepository {

    var syncResult = CompletableDeferred<SyncResult>()
    var syncs = 0

    override suspend fun sync(): SyncResult {
        syncs++
        return syncResult.await()
    }
}
