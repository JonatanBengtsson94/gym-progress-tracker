package com.jonatanbengtsson.gymprogresstracker.ui.start

import androidx.annotation.StringRes
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.appContainer
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkout
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.RefreshResult
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.SyncResult
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsRepository
import kotlinx.coroutines.launch
import java.time.Clock
import java.time.Instant

data class StartWorkoutUiState(
    /** True while the templates are being fetched from the server. */
    val isLoading: Boolean = false,
    /** The templates stored on the device, shown even while they're being fetched or when that fails. */
    val templates: List<WorkoutTemplate> = emptyList(),
    /** Why fetching the templates failed last time. */
    @StringRes val errorMessage: Int? = null,
    /** A workout is in progress once it has an exercise; opening an empty one doesn't count. */
    val workoutInProgress: Boolean = false,
    /** When the workout in progress was started. */
    val workoutStartedAt: Instant? = null,
    /** What the user named the workout in progress, empty until they do. */
    val workoutName: String = "",
    /** How many saved workouts haven't been synced to the server yet. */
    val pendingWorkouts: Int = 0,
    /** True while the saved workouts are being synced. */
    val isSyncing: Boolean = false,
    /** Why syncing failed last time. */
    @StringRes val syncErrorMessage: Int? = null
)

class StartWorkoutViewModel(
    private val templatesRepository: TemplatesRepository,
    private val activeWorkoutRepository: ActiveWorkoutRepository,
    private val sessionRepository: SessionRepository,
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

    /** Sends the saved workouts to the server. */
    fun sync() {
        if (uiState.isSyncing) return
        uiState = uiState.copy(isSyncing = true, syncErrorMessage = null)

        viewModelScope.launch {
            val result = workoutsRepository.sync()
            uiState = uiState.copy(
                isSyncing = false,
                syncErrorMessage = when (result) {
                    SyncResult.NetworkError -> R.string.start_workout_sync_error_network
                    SyncResult.ServerError -> R.string.start_workout_sync_error_server
                    SyncResult.Success, SyncResult.SessionExpired -> null
                }
            )
        }
    }

    /** Starts an empty workout now, unless one is already in progress. */
    fun startNewWorkout() = activeWorkoutRepository.update { workout ->
        if (workout.exercises.isEmpty()) ActiveWorkout(startedAt = clock.instant()) else workout
    }

    /** Throws away the workout in progress. */
    fun discardWorkout() = activeWorkoutRepository.update { ActiveWorkout() }

    fun logOut() {
        viewModelScope.launch { sessionRepository.logOut() }
    }

    companion object {
        val Factory = viewModelFactory {
            initializer {
                StartWorkoutViewModel(
                    appContainer.templatesRepository,
                    appContainer.activeWorkoutRepository,
                    appContainer.sessionRepository,
                    appContainer.workoutsRepository
                )
            }
        }
    }
}
