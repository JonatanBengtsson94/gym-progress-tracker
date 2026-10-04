package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutDao
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutExerciseEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutExerciseWithSets
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutSetEntity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.time.Instant

/**
 * A set as typed, kept as text so partial input like "62," survives until it's finished. [id] tells the
 * set apart from the others in its exercise, so it can be followed as sets before it are removed.
 */
data class SetEntry(
    val weightKg: String = "",
    val reps: String = "",
    val completed: Boolean = false,
    val id: Int = 0
) {
    /** A set can only be completed once it has at least one rep. */
    val canComplete: Boolean get() = (reps.toIntOrNull() ?: 0) > 0
}

data class WorkoutExerciseEntry(val exercise: Exercise, val sets: List<SetEntry> = listOf(SetEntry())) {
    val allSetsCompleted: Boolean get() = sets.isNotEmpty() && sets.all { it.completed }

    /** True when any set has a weight, reps or is completed, i.e. removing the exercise would lose input. */
    val hasEnteredSets: Boolean get() = sets.any { it.weightKg.isNotEmpty() || it.reps.isNotEmpty() || it.completed }
}

data class ActiveWorkout(
    /** When the workout was started, or null when none has been. */
    val startedAt: Instant? = null,
    /** What the user named the workout, empty until they do. */
    val name: String = "",
    /** The exercises, in the order they were added. */
    val exercises: List<WorkoutExerciseEntry> = emptyList()
)

/** The workout being logged, kept on the device so it outlives both the screen and the app's process. */
interface ActiveWorkoutRepository {
    /** The workout being logged, an empty [ActiveWorkout] when there is none, and null until it has been read from disk. */
    val workout: StateFlow<ActiveWorkout?>

    /** Replaces the workout with [transform] applied to it, and saves the result. Ignored until the workout has loaded. */
    fun update(transform: (ActiveWorkout) -> ActiveWorkout)
}

/**
 * Keeps the workout in memory, so updates show at once, and saves it to Room in [externalScope], so a
 * save outlives the screen that made it. After a burst of updates only the latest workout is written.
 */
class RoomActiveWorkoutRepository(
    private val dao: ActiveWorkoutDao,
    externalScope: CoroutineScope
) : ActiveWorkoutRepository {

    private val _workout = MutableStateFlow<ActiveWorkout?>(null)
    override val workout: StateFlow<ActiveWorkout?> = _workout.asStateFlow()

    init {
        externalScope.launch {
            val saved = dao.getWorkout()
            _workout.value = ActiveWorkout(
                startedAt = saved?.startedAt,
                name = saved?.name.orEmpty(),
                exercises = dao.getExercises().map { it.toEntry() }
            )
            _workout.filterNotNull().collect { save(it) }
        }
    }

    override fun update(transform: (ActiveWorkout) -> ActiveWorkout) {
        _workout.update { it?.let(transform) }
    }

    private suspend fun save(workout: ActiveWorkout) = dao.replaceWorkout(
        workout = ActiveWorkoutEntity(startedAt = workout.startedAt, name = workout.name),
        exercises = workout.exercises.mapIndexed { position, entry ->
            ActiveWorkoutExerciseEntity(exerciseId = entry.exercise.id, name = entry.exercise.name, position = position)
        },
        sets = workout.exercises.flatMap { entry ->
            entry.sets.mapIndexed { position, set ->
                ActiveWorkoutSetEntity(
                    exerciseId = entry.exercise.id,
                    position = position,
                    weightKg = set.weightKg,
                    reps = set.reps,
                    completed = set.completed
                )
            }
        }
    )

    private fun ActiveWorkoutExerciseWithSets.toEntry() = WorkoutExerciseEntry(
        exercise = Exercise(id = exercise.exerciseId, name = exercise.name),
        sets = sets.sortedBy { it.position }.map {
            SetEntry(weightKg = it.weightKg, reps = it.reps, completed = it.completed, id = it.position)
        }
    )
}
