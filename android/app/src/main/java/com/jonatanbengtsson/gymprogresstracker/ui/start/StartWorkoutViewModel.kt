package com.jonatanbengtsson.gymprogresstracker.ui.start

import androidx.annotation.StringRes
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesResult
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import kotlinx.coroutines.launch

data class StartWorkoutUiState(
    val isLoading: Boolean = false,
    val templates: List<WorkoutTemplate> = emptyList(),
    @StringRes val errorMessage: Int? = null,
    val sessionExpired: Boolean = false,
    /** A workout is in progress once it has an exercise; opening an empty one doesn't count. */
    val workoutInProgress: Boolean = false
)

class StartWorkoutViewModel(
    private val templatesApi: TemplatesApi,
    private val activeWorkoutRepository: ActiveWorkoutRepository,
    private val sessionId: String
) : ViewModel() {

    var uiState by mutableStateOf(StartWorkoutUiState())
        private set

    init {
        viewModelScope.launch {
            activeWorkoutRepository.exercises.collect { exercises ->
                uiState = uiState.copy(workoutInProgress = !exercises.isNullOrEmpty())
            }
        }
        loadTemplates()
    }

    fun loadTemplates() {
        if (uiState.isLoading) return
        uiState = uiState.copy(isLoading = true, errorMessage = null)

        viewModelScope.launch {
            uiState = when (val result = templatesApi.getTemplates(sessionId)) {
                is TemplatesResult.Success -> uiState.copy(isLoading = false, templates = result.templates)
                TemplatesResult.SessionExpired -> uiState.copy(isLoading = false, sessionExpired = true)
                TemplatesResult.NetworkError -> uiState.copy(isLoading = false, errorMessage = R.string.start_workout_error_network)
                TemplatesResult.ServerError -> uiState.copy(isLoading = false, errorMessage = R.string.start_workout_error_server)
            }
        }
    }

    /** Throws away the workout in progress. */
    fun discardWorkout() = activeWorkoutRepository.update { emptyList() }
}
