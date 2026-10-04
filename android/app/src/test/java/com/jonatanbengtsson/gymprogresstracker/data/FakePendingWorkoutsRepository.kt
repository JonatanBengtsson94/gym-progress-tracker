package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.update
import kotlin.uuid.Uuid

/** Keeps the workouts in memory, in the order they were added. */
class FakePendingWorkoutsRepository(workouts: List<PendingWorkout> = emptyList()) : PendingWorkoutsRepository {

    override val workouts = MutableStateFlow(workouts)

    override suspend fun add(workout: PendingWorkout) {
        workouts.update { it.filter { pending -> pending.workoutId != workout.workoutId } + workout }
    }

    override suspend fun remove(workoutId: Uuid) {
        workouts.update { it.filter { pending -> pending.workoutId != workoutId } }
    }

    override suspend fun clear() {
        workouts.value = emptyList()
    }
}
