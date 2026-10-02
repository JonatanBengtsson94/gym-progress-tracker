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

/** A set as typed, kept as text so partial input like "62," survives until it's finished. */
data class SetEntry(val weightKg: String = "", val reps: String = "", val completed: Boolean = false) {
    /** A set can only be completed once it has at least one rep. */
    val canComplete: Boolean get() = (reps.toIntOrNull() ?: 0) > 0
}

data class WorkoutExerciseEntry(val exercise: Exercise, val sets: List<SetEntry> = listOf(SetEntry())) {
    val allSetsCompleted: Boolean get() = sets.isNotEmpty() && sets.all { it.completed }
}

data class WorkoutUiState(
    /** The exercises added to the workout, in the order they were added. */
    val workoutExercises: List<WorkoutExerciseEntry> = emptyList(),
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

    /** Adds [exercise] to the end of the workout with one empty set, unless it's already in it. */
    fun addExercise(exercise: Exercise) {
        if (uiState.workoutExercises.any { it.exercise.id == exercise.id }) return
        uiState = uiState.copy(workoutExercises = uiState.workoutExercises + WorkoutExerciseEntry(exercise))
    }

    /** Adds an uncompleted set to the exercise, prefilled with its last set's weight and reps. */
    fun addSet(exerciseId: Long) = updateSets(exerciseId) { sets ->
        sets + (sets.lastOrNull()?.copy(completed = false) ?: SetEntry())
    }

    /** Toggles whether the set is completed. Ignored for a set that [SetEntry.canComplete] rules out. */
    fun toggleSetCompleted(exerciseId: Long, setIndex: Int) = updateSet(exerciseId, setIndex) { set ->
        if (set.completed || set.canComplete) set.copy(completed = !set.completed) else set
    }

    fun removeSet(exerciseId: Long, setIndex: Int) = updateSets(exerciseId) { sets ->
        sets.filterIndexed { index, _ -> index != setIndex }
    }

    /** Ignores input that isn't a weight in kg with at most two decimals. */
    fun updateWeight(exerciseId: Long, setIndex: Int, weightKg: String) {
        if (!WEIGHT_INPUT.matches(weightKg)) return
        updateSet(exerciseId, setIndex) { it.copy(weightKg = weightKg) }
    }

    /** Ignores input that isn't a whole number of reps. Reps that no longer allow completion un-complete the set. */
    fun updateReps(exerciseId: Long, setIndex: Int, reps: String) {
        if (!REPS_INPUT.matches(reps)) return
        updateSet(exerciseId, setIndex) { set ->
            val updated = set.copy(reps = reps)
            if (updated.canComplete) updated else updated.copy(completed = false)
        }
    }

    private fun updateSet(exerciseId: Long, setIndex: Int, transform: (SetEntry) -> SetEntry) =
        updateSets(exerciseId) { sets ->
            sets.mapIndexed { index, set -> if (index == setIndex) transform(set) else set }
        }

    private fun updateSets(exerciseId: Long, transform: (List<SetEntry>) -> List<SetEntry>) {
        uiState = uiState.copy(
            workoutExercises = uiState.workoutExercises.map { entry ->
                if (entry.exercise.id == exerciseId) entry.copy(sets = transform(entry.sets)) else entry
            }
        )
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

    private companion object {
        // Accepts a decimal comma as well as a point, since keyboards in many locales offer only one.
        val WEIGHT_INPUT = Regex("""\d{0,4}([.,]\d{0,2})?""")
        // The backend stores reps as a uint8, so three digits is the most it can take.
        val REPS_INPUT = Regex("""\d{0,3}""")
    }
}
