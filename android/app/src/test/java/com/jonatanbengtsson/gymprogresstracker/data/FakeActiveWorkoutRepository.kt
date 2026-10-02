package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.update

/** Keeps the workout in memory only. Set [exercises] to null to act as if it hasn't loaded yet. */
class FakeActiveWorkoutRepository(exercises: List<WorkoutExerciseEntry>? = emptyList()) : ActiveWorkoutRepository {

    override val exercises = MutableStateFlow(exercises)

    override fun update(transform: (List<WorkoutExerciseEntry>) -> List<WorkoutExerciseEntry>) {
        this.exercises.update { it?.let(transform) }
    }
}
