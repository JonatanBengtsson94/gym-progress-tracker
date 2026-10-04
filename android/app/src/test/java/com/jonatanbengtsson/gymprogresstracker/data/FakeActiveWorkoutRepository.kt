package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.update

/** Keeps the workout in memory only. Set [workout] to null to act as if it hasn't loaded yet. */
class FakeActiveWorkoutRepository(workout: ActiveWorkout? = ActiveWorkout()) : ActiveWorkoutRepository {

    override val workout = MutableStateFlow(workout)

    override fun update(transform: (ActiveWorkout) -> ActiveWorkout) {
        this.workout.update { it?.let(transform) }
    }
}
