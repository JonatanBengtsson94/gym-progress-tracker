package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutDao
import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutSetEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.PendingWorkoutWithSets
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.Instant
import kotlin.uuid.Uuid

class RoomPendingWorkoutsRepositoryTest {

    /** Stores the workouts in memory, handing sets back in reverse since Room doesn't promise their order. */
    private class FakePendingWorkoutDao : PendingWorkoutDao {
        val workouts = MutableStateFlow(emptyList<PendingWorkoutEntity>())
        var sets = emptyList<PendingWorkoutSetEntity>()

        override fun observeWorkouts() = workouts.map { workouts ->
            workouts.sortedBy { it.completedAt }.map { workout ->
                PendingWorkoutWithSets(workout, sets.filter { it.workoutId == workout.workoutId }.reversed())
            }
        }

        override suspend fun replaceWorkout(workout: PendingWorkoutEntity, sets: List<PendingWorkoutSetEntity>) {
            deleteWorkout(workout.workoutId)
            this.sets += sets
            workouts.value += workout
        }

        override suspend fun deleteWorkout(workoutId: Uuid) {
            sets = sets.filter { it.workoutId != workoutId }
            workouts.value = workouts.value.filter { it.workoutId != workoutId }
        }

        override suspend fun deleteWorkouts() {
            sets = emptyList()
            workouts.value = emptyList()
        }

        override suspend fun insertWorkout(workout: PendingWorkoutEntity) = error("Stored as a whole by replaceWorkout")
        override suspend fun insertSets(sets: List<PendingWorkoutSetEntity>) = error("Stored as a whole by replaceWorkout")
    }

    private val dao = FakePendingWorkoutDao()
    private val repository = RoomPendingWorkoutsRepository(dao)

    private val workout = PendingWorkout(
        workoutId = testId(20),
        templateId = testId(10),
        templateIsNew = true,
        workout = FinishedWorkout(
            name = "Push day",
            startedAt = Instant.parse("2026-10-04T17:00:00Z"),
            completedAt = Instant.parse("2026-10-04T18:00:00Z"),
            exercises = listOf(
                WorkoutExercise(testId(1), "Squat (Barbell)", listOf(WorkoutSet(5, 100000), WorkoutSet(3, 102500))),
                WorkoutExercise(testId(2), "Bench Press (Barbell)", listOf(WorkoutSet(8, 60000)))
            )
        )
    )

    @Test
    fun `an added workout is stored with its sets in position`() = runTest {
        repository.add(workout)

        assertEquals(
            listOf(PendingWorkoutEntity(testId(20), testId(10), true, "Push day", workout.workout.startedAt, workout.workout.completedAt)),
            dao.workouts.value
        )
        assertEquals(
            listOf(
                PendingWorkoutSetEntity(testId(20), 0, testId(1), "Squat (Barbell)", 5, 100000),
                PendingWorkoutSetEntity(testId(20), 1, testId(1), "Squat (Barbell)", 3, 102500),
                PendingWorkoutSetEntity(testId(20), 2, testId(2), "Bench Press (Barbell)", 8, 60000)
            ),
            dao.sets
        )
    }

    @Test
    fun `a stored workout is read back with its exercises in order`() = runTest {
        repository.add(workout)

        assertEquals(listOf(workout), repository.workouts.first())
    }

    @Test
    fun `a workout under a template already on the server is read back as such`() = runTest {
        val underExisting = workout.copy(templateIsNew = false)

        repository.add(underExisting)

        assertEquals(listOf(underExisting), repository.workouts.first())
    }

    @Test
    fun `workouts are read back oldest first`() = runTest {
        val earlier = workout.copy(workoutId = testId(21), workout = workout.workout.copy(completedAt = Instant.parse("2026-10-03T18:00:00Z")))

        repository.add(workout)
        repository.add(earlier)

        assertEquals(listOf(earlier, workout), repository.workouts.first())
    }

    @Test
    fun `removing a workout keeps the others`() = runTest {
        val other = workout.copy(workoutId = testId(21))
        repository.add(workout)
        repository.add(other)

        repository.remove(testId(20))

        assertEquals(listOf(other), repository.workouts.first())
    }

    @Test
    fun `clearing removes every workout`() = runTest {
        repository.add(workout)
        repository.add(workout.copy(workoutId = testId(21)))

        repository.clear()

        assertEquals(emptyList<PendingWorkout>(), repository.workouts.first())
    }
}
