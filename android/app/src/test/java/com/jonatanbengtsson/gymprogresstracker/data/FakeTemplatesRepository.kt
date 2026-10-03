package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.flow.MutableStateFlow

/** Keeps the templates in memory. Each refresh suspends until the test completes [refreshResult]. */
class FakeTemplatesRepository(templates: List<WorkoutTemplate> = emptyList()) : TemplatesRepository {

    override val templates = MutableStateFlow(templates)
    var refreshResult = CompletableDeferred<RefreshResult>()
    val refreshes = mutableListOf<String>()

    override suspend fun refresh(sessionId: String): RefreshResult {
        refreshes += sessionId
        return refreshResult.await()
    }
}
