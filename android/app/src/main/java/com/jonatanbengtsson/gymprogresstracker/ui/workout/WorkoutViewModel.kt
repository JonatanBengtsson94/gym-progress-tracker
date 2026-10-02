package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.annotation.StringRes
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesApi
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesResult
import kotlinx.coroutines.launch

data class WorkoutUiState(
    /** The exercises added to the workout, in the order they were added. */
    val workoutExercises: List<Exercise> = emptyList(),
    val isLoadingExercises: Boolean = false,
    val exercises: List<Exercise> = emptyList(),
    @StringRes val exercisesErrorMessage: Int? = null,
    val sessionExpired: Boolean = false
)

class WorkoutViewModel(
    private val exercisesApi: ExercisesApi,
    private val sessionId: String
) : ViewModel() {

    var uiState by mutableStateOf(WorkoutUiState())
        private set

    // Loaded up front so the list is ready by the time the user adds an exercise.
    init {
        loadExercises()
    }

    /** Adds [exercise] to the end of the workout, unless it's already in it. */
    fun addExercise(exercise: Exercise) {
        if (uiState.workoutExercises.any { it.id == exercise.id }) return
        uiState = uiState.copy(workoutExercises = uiState.workoutExercises + exercise)
    }

    fun loadExercises() {
        if (uiState.isLoadingExercises) return
        uiState = uiState.copy(isLoadingExercises = true, exercisesErrorMessage = null)

        viewModelScope.launch {
            uiState = when (val result = exercisesApi.getExercises(sessionId)) {
                is ExercisesResult.Success -> uiState.copy(isLoadingExercises = false, exercises = result.exercises)
                ExercisesResult.SessionExpired -> uiState.copy(isLoadingExercises = false, sessionExpired = true)
                ExercisesResult.NetworkError -> uiState.copy(isLoadingExercises = false, exercisesErrorMessage = R.string.workout_exercises_error_network)
                ExercisesResult.ServerError -> uiState.copy(isLoadingExercises = false, exercisesErrorMessage = R.string.workout_exercises_error_server)
            }
        }
    }
}
