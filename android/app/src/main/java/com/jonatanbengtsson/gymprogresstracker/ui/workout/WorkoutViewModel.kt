package com.jonatanbengtsson.gymprogresstracker.ui.workout

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
import com.jonatanbengtsson.gymprogresstracker.data.ApiResult
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.FinishedExercise
import com.jonatanbengtsson.gymprogresstracker.data.FinishedWorkout
import com.jonatanbengtsson.gymprogresstracker.data.RefreshResult
import com.jonatanbengtsson.gymprogresstracker.data.SetEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutSet
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsRepository
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.launch
import java.time.Clock
import java.time.Instant
import kotlin.uuid.Uuid

data class WorkoutUiState(
    /** True until the saved workout has been read from disk. */
    val isLoadingWorkout: Boolean = false,
    /** When the workout was started, or null when it hasn't been. */
    val startedAt: Instant? = null,
    /** What the user named the workout, empty until they do. */
    val name: String = "",
    /** The exercises added to the workout, in the order they were added. */
    val workoutExercises: List<WorkoutExerciseEntry> = emptyList(),
    /** True while the exercises to pick from are being fetched from the server. */
    val isLoadingExercises: Boolean = false,
    /** The exercises stored on the device, shown even while they're being fetched or when that fails. */
    val exercises: List<Exercise> = emptyList(),
    /** Why fetching the exercises failed last time. */
    @StringRes val exercisesErrorMessage: Int? = null,
    /** True while the workout is being saved to the server. */
    val isSaving: Boolean = false,
    /** Why the workout couldn't be saved last time. */
    @StringRes val saveErrorMessage: Int? = null,
    /** True once the workout has been saved and cleared to make way for the next one. */
    val isSaved: Boolean = false
)

class WorkoutViewModel(
    private val exercisesRepository: ExercisesRepository,
    private val activeWorkoutRepository: ActiveWorkoutRepository,
    private val workoutsRepository: WorkoutsRepository,
    private val clock: Clock = Clock.systemUTC()
) : ViewModel() {

    var uiState by mutableStateOf(WorkoutUiState(isLoadingWorkout = true))
        private set

    init {
        // viewModelScope runs on Dispatchers.Main.immediate, so an edit reaches uiState before the next
        // keystroke arrives; the set text fields would drop input otherwise.
        viewModelScope.launch {
            activeWorkoutRepository.workout.filterNotNull().collect { workout ->
                uiState = uiState.copy(
                    isLoadingWorkout = false,
                    startedAt = workout.startedAt,
                    name = workout.name,
                    workoutExercises = workout.exercises
                )
            }
        }
        viewModelScope.launch {
            exercisesRepository.exercises.collect { exercises -> uiState = uiState.copy(exercises = exercises) }
        }
        loadExercises()
    }

    /** Ignores names longer than a template name can be. */
    fun updateName(name: String) {
        if (name.length > MAX_NAME_LENGTH) return
        activeWorkoutRepository.update { it.copy(name = name) }
    }

    /** Adds [exercise] to the end of the workout with one empty set, unless it's already in it. */
    fun addExercise(exercise: Exercise) = updateExercises { exercises ->
        if (exercises.any { it.exercise.id == exercise.id }) exercises else exercises + WorkoutExerciseEntry(exercise)
    }

    fun removeExercise(exerciseId: Uuid) = updateExercises { exercises ->
        exercises.filter { it.exercise.id != exerciseId }
    }

    /** Adds an uncompleted set to the exercise, prefilled with its last set's weight and reps. */
    fun addSet(exerciseId: Uuid) = updateSets(exerciseId) { sets ->
        val id = (sets.maxOfOrNull { it.id } ?: -1) + 1
        sets + (sets.lastOrNull()?.copy(completed = false, id = id) ?: SetEntry(id = id))
    }

    /** Toggles whether the set is completed. Ignored for a set that [SetEntry.canComplete] rules out. */
    fun toggleSetCompleted(exerciseId: Uuid, setIndex: Int) = updateSet(exerciseId, setIndex) { set ->
        if (set.completed || set.canComplete) set.copy(completed = !set.completed) else set
    }

    fun removeSet(exerciseId: Uuid, setIndex: Int) = updateSets(exerciseId) { sets ->
        sets.filterIndexed { index, _ -> index != setIndex }
    }

    /** Ignores input that isn't a weight in kg with at most two decimals. */
    fun updateWeight(exerciseId: Uuid, setIndex: Int, weightKg: String) {
        if (!WEIGHT_INPUT.matches(weightKg)) return
        updateSet(exerciseId, setIndex) { it.copy(weightKg = weightKg) }
    }

    /** Ignores input that isn't a whole number of reps. Reps that no longer allow completion un-complete the set. */
    fun updateReps(exerciseId: Uuid, setIndex: Int, reps: String) {
        if (!REPS_INPUT.matches(reps)) return
        updateSet(exerciseId, setIndex) { set ->
            val updated = set.copy(reps = reps)
            if (updated.canComplete) updated else updated.copy(completed = false)
        }
    }

    private fun updateSet(exerciseId: Uuid, setIndex: Int, transform: (SetEntry) -> SetEntry) =
        updateSets(exerciseId) { sets ->
            sets.mapIndexed { index, set -> if (index == setIndex) transform(set) else set }
        }

    private fun updateSets(exerciseId: Uuid, transform: (List<SetEntry>) -> List<SetEntry>) =
        updateExercises { exercises ->
            exercises.map { entry ->
                if (entry.exercise.id == exerciseId) entry.copy(sets = transform(entry.sets)) else entry
            }
        }

    private fun updateExercises(transform: (List<WorkoutExerciseEntry>) -> List<WorkoutExerciseEntry>) =
        activeWorkoutRepository.update { it.copy(exercises = transform(it.exercises)) }

    /**
     * Saves the workout's completed sets to the server, as completed now, and clears the workout once
     * it's saved. Sets that aren't completed are left out. A workout changed while it's being saved is
     * kept instead, so the changes can be saved too.
     */
    fun saveWorkout() {
        if (uiState.isSaving) return
        activeWorkoutRepository.update { it.copy(workoutId = it.workoutId ?: Uuid.random()) }
        val workout = activeWorkoutRepository.workout.value ?: return
        val workoutId = workout.workoutId ?: return

        val exercises = workout.exercises.mapNotNull { entry ->
            val sets = entry.sets.filter { it.completed }.map { WorkoutSet(reps = it.reps.toInt(), weightGrams = it.weightGrams) }
            if (sets.isEmpty()) null else FinishedExercise(entry.exercise.id, sets)
        }
        val invalidMessage = when {
            workout.name.isBlank() -> R.string.workout_save_error_name
            exercises.isEmpty() -> R.string.workout_save_error_no_sets
            else -> null
        }
        uiState = uiState.copy(isSaving = invalidMessage == null, saveErrorMessage = invalidMessage)
        if (invalidMessage != null) return

        val completedAt = clock.instant()
        val finished = FinishedWorkout(
            templateId = null,
            templateName = workout.name.trim(),
            startedAt = workout.startedAt ?: completedAt,
            completedAt = completedAt,
            exercises = exercises
        )
        viewModelScope.launch {
            val result = workoutsRepository.save(workoutId, finished)
            val cleared = result is ApiResult.Success && activeWorkoutRepository.workout.value == workout
            if (cleared) activeWorkoutRepository.update { ActiveWorkout() }
            uiState = uiState.copy(
                isSaving = false,
                isSaved = cleared,
                saveErrorMessage = when (result) {
                    ApiResult.NetworkError -> R.string.workout_save_error_network
                    ApiResult.ServerError -> R.string.workout_save_error_server
                    is ApiResult.Success, ApiResult.Unauthorized -> null
                }
            )
        }
    }

    /** Fetches the exercises from the server, replacing the stored ones. */
    fun loadExercises() {
        if (uiState.isLoadingExercises) return
        uiState = uiState.copy(isLoadingExercises = true, exercisesErrorMessage = null)

        viewModelScope.launch {
            val result = exercisesRepository.refresh()
            uiState = uiState.copy(
                isLoadingExercises = false,
                exercisesErrorMessage = when (result) {
                    RefreshResult.NetworkError -> R.string.workout_exercises_error_network
                    RefreshResult.ServerError -> R.string.workout_exercises_error_server
                    RefreshResult.Success, RefreshResult.SessionExpired -> null
                }
            )
        }
    }

    companion object {
        val Factory = viewModelFactory {
            initializer {
                WorkoutViewModel(appContainer.exercisesRepository, appContainer.activeWorkoutRepository, appContainer.workoutsRepository)
            }
        }

        private val WEIGHT_INPUT = Regex("""\d{0,4}([.,]\d{0,2})?""")
        // The backend stores template names as a VARCHAR(100).
        private const val MAX_NAME_LENGTH = 100
        // The backend stores reps as a uint8, so three digits is the most it can take.
        private val REPS_INPUT = Regex("""\d{0,3}""")
    }
}
