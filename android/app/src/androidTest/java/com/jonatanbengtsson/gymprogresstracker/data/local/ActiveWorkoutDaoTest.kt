package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.jonatanbengtsson.gymprogresstracker.data.testId
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import java.time.Instant

@RunWith(AndroidJUnit4::class)
class ActiveWorkoutDaoTest {

    private lateinit var database: GymDatabase
    private lateinit var dao: ActiveWorkoutDao

    private val workout = ActiveWorkoutEntity(startedAt = Instant.parse("2026-10-04T17:00:00Z"))
    private val squat = ActiveWorkoutExerciseEntity(exerciseId = testId(1), name = "Squat (Barbell)", position = 0)
    private val benchPress = ActiveWorkoutExerciseEntity(exerciseId = testId(2), name = "Bench Press (Barbell)", position = 1)
    private val squatSets = listOf(
        ActiveWorkoutSetEntity(exerciseId = testId(1), position = 0, weightKg = "100", reps = "5", completed = true),
        ActiveWorkoutSetEntity(exerciseId = testId(1), position = 1, weightKg = "102,5", reps = "3", completed = false)
    )
    private val benchPressSet = ActiveWorkoutSetEntity(exerciseId = testId(2), position = 0, weightKg = "60", reps = "8", completed = false)

    @Before
    fun setUp() {
        database = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), GymDatabase::class.java).build()
        dao = database.activeWorkoutDao()
    }

    @After
    fun tearDown() {
        database.close()
    }

    @Test
    fun aNewDatabaseHasNoWorkout() = runTest {
        assertNull(dao.getWorkout())
        assertEquals(emptyList<ActiveWorkoutExerciseWithSets>(), dao.getExercises())
    }

    @Test
    fun aSavedWorkoutIsReadBackWithItsExercisesInPosition() = runTest {
        dao.replaceWorkout(workout, listOf(benchPress, squat), squatSets + benchPressSet)

        val exercises = dao.getExercises()

        assertEquals(workout, dao.getWorkout())
        assertEquals(listOf(squat, benchPress), exercises.map { it.exercise })
        assertEquals(squatSets, exercises[0].sets.sortedBy { it.position })
        assertEquals(listOf(benchPressSet), exercises[1].sets)
    }

    @Test
    fun replacingTheWorkoutLeavesNothingOfTheOldOne() = runTest {
        dao.replaceWorkout(workout, listOf(squat, benchPress), squatSets + benchPressSet)

        val restarted = ActiveWorkoutEntity(startedAt = Instant.parse("2026-10-05T17:00:00Z"))
        dao.replaceWorkout(restarted, listOf(squat), listOf(squatSets[0]))

        assertEquals(restarted, dao.getWorkout())
        assertEquals(listOf(ActiveWorkoutExerciseWithSets(squat, listOf(squatSets[0]))), dao.getExercises())
    }

    @Test
    fun anEmptyWorkoutReplacesTheSavedOne() = runTest {
        dao.replaceWorkout(workout, listOf(squat), squatSets)

        dao.replaceWorkout(null, emptyList(), emptyList())

        assertNull(dao.getWorkout())
        assertEquals(emptyList<ActiveWorkoutExerciseWithSets>(), dao.getExercises())
    }
}
