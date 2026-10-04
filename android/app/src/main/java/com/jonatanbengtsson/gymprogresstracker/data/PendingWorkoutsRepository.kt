package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutDao
import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutSetEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutWithSets
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlin.uuid.Uuid

/**
 * A workout saved on the device but not yet sent to the server, logged under the template [templateId].
 * [templateIsNew] is true when that template was created on the device and may not be on the server yet.
 */
data class PendingWorkout(
    val workoutId: Uuid,
    val templateId: Uuid,
    val templateIsNew: Boolean,
    val workout: FinishedWorkout
)

/** Finished workouts kept on the device until they've been sent to the server. */
interface PendingWorkoutsRepository {
    /** Oldest first, by when they were completed. */
    val workouts: Flow<List<PendingWorkout>>

    /** Adds [workout], replacing any with the same id. */
    suspend fun add(workout: PendingWorkout)

    suspend fun remove(workoutId: Uuid)

    suspend fun clear()
}

class RoomPendingWorkoutsRepository(private val dao: PendingWorkoutDao) : PendingWorkoutsRepository {

    override val workouts: Flow<List<PendingWorkout>> =
        dao.observeWorkouts().map { workouts -> workouts.map { it.toPendingWorkout() } }

    override suspend fun add(workout: PendingWorkout) = dao.replaceWorkout(
        workout = with(workout) {
            PendingWorkoutEntity(workoutId, templateId, templateIsNew, this.workout.name, this.workout.startedAt, this.workout.completedAt)
        },
        sets = workout.workout.exercises
            .flatMap { exercise -> exercise.sets.map { set -> exercise to set } }
            .mapIndexed { position, (exercise, set) ->
                PendingWorkoutSetEntity(workout.workoutId, position, exercise.exerciseId, exercise.name, set.reps, set.weightGrams)
            }
    )

    override suspend fun remove(workoutId: Uuid) = dao.deleteWorkout(workoutId)

    override suspend fun clear() = dao.deleteWorkouts()

    /** Sets were stored in workout order, which is grouped by exercise. */
    private fun PendingWorkoutWithSets.toPendingWorkout() = PendingWorkout(
        workoutId = workout.workoutId,
        templateId = workout.templateId,
        templateIsNew = workout.templateIsNew,
        workout = FinishedWorkout(
            name = workout.name,
            startedAt = workout.startedAt,
            completedAt = workout.completedAt,
            exercises = sets.sortedBy { it.position }
                .groupBy { it.exerciseId }
                .map { (exerciseId, sets) ->
                    WorkoutExercise(exerciseId, sets.first().exerciseName, sets.map { WorkoutSet(it.reps, it.weightGrams) })
                }
        )
    )
}
