package com.jonatanbengtsson.gymprogresstracker.ui.start

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.jonatanbengtsson.gymprogresstracker.appContainer
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkout
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsRepository
import kotlinx.coroutines.launch
import java.time.Clock
import java.time.Instant

data class StartWorkoutUiState(
    /** The templates stored on the device. */
    val templates: List<WorkoutTemplate> = emptyList(),
    /** A workout is in progress once it has an exercise; opening an empty one doesn't count. */
    val workoutInProgress: Boolean = false,
    /** When the workout in progress was started. */
    val workoutStartedAt: Instant? = null,
    /** What the user named the workout in progress, empty until they do. */
    val workoutName: String = "",
    /** How many saved workouts haven't been synced to the server yet. */
    val pendingWorkouts: Int = 0
)

class StartWorkoutViewModel(
    private val templatesRepository: TemplatesRepository,
    private val activeWorkoutRepository: ActiveWorkoutRepository,
    private val workoutsRepository: WorkoutsRepository,
    private val clock: Clock = Clock.systemUTC()
) : ViewModel() {

    var uiState by mutableStateOf(StartWorkoutUiState())
        private set

    init {
        viewModelScope.launch {
            activeWorkoutRepository.workout.collect { workout ->
                uiState = uiState.copy(
                    workoutInProgress = !workout?.exercises.isNullOrEmpty(),
                    workoutStartedAt = workout?.startedAt,
                    workoutName = workout?.name.orEmpty()
                )
            }
        }
        viewModelScope.launch {
            templatesRepository.templates.collect { templates -> uiState = uiState.copy(templates = templates) }
        }
        viewModelScope.launch {
            workoutsRepository.pendingCount.collect { count -> uiState = uiState.copy(pendingWorkouts = count) }
        }
    }

    /** Starts an empty workout now, unless one is already in progress. */
    fun startNewWorkout() = activeWorkoutRepository.update { workout ->
        if (workout.exercises.isEmpty()) ActiveWorkout(startedAt = clock.instant()) else workout
    }

    /** Throws away the workout in progress. */
    fun discardWorkout() = activeWorkoutRepository.update { ActiveWorkout() }

    companion object {
        val Factory = viewModelFactory {
            initializer {
                StartWorkoutViewModel(
                    appContainer.templatesRepository,
                    appContainer.activeWorkoutRepository,
                    appContainer.workoutsRepository
                )
            }
        }
    }
}
