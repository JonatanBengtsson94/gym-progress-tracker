package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutDao
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutExerciseEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutExerciseWithSets
import com.jonatanbengtsson.gymprogresstracker.data.local.ActiveWorkoutSetEntity
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.time.Instant

@OptIn(ExperimentalCoroutinesApi::class)
class RoomActiveWorkoutRepositoryTest {

    /** Stores the workout in memory, handing sets back in reverse since Room doesn't promise their order. */
    private class FakeActiveWorkoutDao : ActiveWorkoutDao {
        var workout: ActiveWorkoutEntity? = null
        var exercises = emptyList<ActiveWorkoutExerciseEntity>()
        var sets = emptyList<ActiveWorkoutSetEntity>()
        var saves = 0

        override suspend fun getWorkout() = workout

        override suspend fun getExercises() = exercises.sortedBy { it.position }.map { exercise ->
            ActiveWorkoutExerciseWithSets(exercise, sets.filter { it.exerciseId == exercise.exerciseId }.reversed())
        }

        override suspend fun replaceWorkout(
            workout: ActiveWorkoutEntity,
            exercises: List<ActiveWorkoutExerciseEntity>,
            sets: List<ActiveWorkoutSetEntity>
        ) {
            this.workout = workout
            this.exercises = exercises
            this.sets = sets
            saves++
        }

        override suspend fun insertWorkout(workout: ActiveWorkoutEntity) = error("Replaced as a whole by replaceWorkout")
        override suspend fun deleteExercises() = error("Replaced as a whole by replaceWorkout")
        override suspend fun insertExercises(exercises: List<ActiveWorkoutExerciseEntity>) = error("Replaced as a whole by replaceWorkout")
        override suspend fun insertSets(sets: List<ActiveWorkoutSetEntity>) = error("Replaced as a whole by replaceWorkout")
    }

    private val squat = Exercise(testId(1), "Squat (Barbell)")
    private val benchPress = Exercise(testId(2), "Bench Press (Barbell)")

    private val startedAt = Instant.parse("2026-10-04T17:00:00Z")

    private val workout = ActiveWorkout(
        startedAt = startedAt,
        name = "Push day",
        exercises = listOf(
            WorkoutExerciseEntry(squat, listOf(SetEntry("100", "5", completed = true), SetEntry("102,5", "3", id = 1))),
            WorkoutExerciseEntry(benchPress, listOf(SetEntry()))
        )
    )

    private val dao = FakeActiveWorkoutDao()

    private fun TestScope.repository() = RoomActiveWorkoutRepository(dao, backgroundScope)

    /** Saves [workout] the way an earlier run of the app would have. */
    private fun TestScope.givenSaved(workout: ActiveWorkout) {
        val earlier = repository()
        runCurrent()
        earlier.update { workout }
        runCurrent()
    }

    @Test
    fun `the workout is null until the saved one has been read`() = runTest {
        val repository = repository()
        assertNull(repository.workout.value)

        runCurrent()

        assertEquals(ActiveWorkout(), repository.workout.value)
    }

    @Test
    fun `a saved workout is read back in order`() = runTest {
        givenSaved(workout)

        val repository = repository()
        runCurrent()

        assertEquals(workout, repository.workout.value)
    }

    @Test
    fun `a started workout without exercises is read back`() = runTest {
        givenSaved(ActiveWorkout(startedAt, name = "Push day"))

        val repository = repository()
        runCurrent()

        assertEquals(ActiveWorkout(startedAt, name = "Push day"), repository.workout.value)
    }

    @Test
    fun `an update shows at once and is saved with each exercise and set in position`() = runTest {
        val repository = repository()
        runCurrent()

        repository.update { workout }
        assertEquals(workout, repository.workout.value)
        runCurrent()

        assertEquals(ActiveWorkoutEntity(startedAt = startedAt, name = "Push day"), dao.workout)
        assertEquals(
            listOf(ActiveWorkoutExerciseEntity(testId(1), "Squat (Barbell)", 0), ActiveWorkoutExerciseEntity(testId(2), "Bench Press (Barbell)", 1)),
            dao.exercises
        )
        assertEquals(
            listOf(
                ActiveWorkoutSetEntity(testId(1), 0, "100", "5", completed = true),
                ActiveWorkoutSetEntity(testId(1), 1, "102,5", "3", completed = false),
                ActiveWorkoutSetEntity(testId(2), 0, "", "", completed = false)
            ),
            dao.sets
        )
    }

    @Test
    fun `a burst of updates saves only the latest workout`() = runTest {
        val repository = repository()
        runCurrent()
        val savesBefore = dao.saves

        repository.update { ActiveWorkout(startedAt, exercises = listOf(WorkoutExerciseEntry(squat))) }
        repository.update { it.copy(exercises = it.exercises + WorkoutExerciseEntry(benchPress)) }
        repository.update { it.copy(exercises = it.exercises.drop(1)) }
        runCurrent()

        assertEquals(savesBefore + 1, dao.saves)
        assertEquals(listOf(ActiveWorkoutExerciseEntity(testId(2), "Bench Press (Barbell)", 0)), dao.exercises)
    }

    @Test
    fun `updates before the saved workout has been read are ignored`() = runTest {
        givenSaved(workout)
        val repository = repository()

        repository.update { ActiveWorkout() }
        runCurrent()

        assertEquals(workout, repository.workout.value)
    }

    @Test
    fun `an emptied workout is saved as not started and unnamed`() = runTest {
        givenSaved(workout)
        val repository = repository()
        runCurrent()

        repository.update { ActiveWorkout() }
        runCurrent()

        assertEquals(ActiveWorkoutEntity(startedAt = null, name = ""), dao.workout)
        assertEquals(emptyList<ActiveWorkoutExerciseEntity>(), dao.exercises)
    }
}
