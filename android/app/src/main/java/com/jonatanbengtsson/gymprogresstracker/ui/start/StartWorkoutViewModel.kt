package com.jonatanbengtsson.gymprogresstracker.ui.start

import androidx.annotation.StringRes
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.RefreshResult
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import kotlinx.coroutines.launch

data class StartWorkoutUiState(
    /** True while the templates are being fetched from the server. */
    val isLoading: Boolean = false,
    /** The templates stored on the device, shown even while they're being fetched or when that fails. */
    val templates: List<WorkoutTemplate> = emptyList(),
    /** Why fetching the templates failed last time. */
    @StringRes val errorMessage: Int? = null,
    /** A workout is in progress once it has an exercise; opening an empty one doesn't count. */
    val workoutInProgress: Boolean = false
)

class StartWorkoutViewModel(
    private val templatesRepository: TemplatesRepository,
    private val activeWorkoutRepository: ActiveWorkoutRepository,
    private val sessionRepository: SessionRepository
) : ViewModel() {

    var uiState by mutableStateOf(StartWorkoutUiState())
        private set

    init {
        viewModelScope.launch {
            activeWorkoutRepository.exercises.collect { exercises ->
                uiState = uiState.copy(workoutInProgress = !exercises.isNullOrEmpty())
            }
        }
        viewModelScope.launch {
            templatesRepository.templates.collect { templates -> uiState = uiState.copy(templates = templates) }
        }
        loadTemplates()
    }

    /** Fetches the templates from the server, replacing the stored ones. */
    fun loadTemplates() {
        if (uiState.isLoading) return
        uiState = uiState.copy(isLoading = true, errorMessage = null)

        viewModelScope.launch {
            val result = templatesRepository.refresh()
            uiState = uiState.copy(
                isLoading = false,
                errorMessage = when (result) {
                    RefreshResult.NetworkError -> R.string.start_workout_error_network
                    RefreshResult.ServerError -> R.string.start_workout_error_server
                    RefreshResult.Success, RefreshResult.SessionExpired -> null
                }
            )
        }
    }

    /** Throws away the workout in progress. */
    fun discardWorkout() = activeWorkoutRepository.update { emptyList() }

    fun logOut() {
        viewModelScope.launch { sessionRepository.logOut() }
    }
}
