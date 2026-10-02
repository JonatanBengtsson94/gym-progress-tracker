package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class ActiveWorkoutDaoTest {

    private lateinit var database: GymDatabase
    private lateinit var dao: ActiveWorkoutDao

    private val squat = ActiveWorkoutExerciseEntity(exerciseId = 1, name = "Squat (Barbell)", position = 0)
    private val benchPress = ActiveWorkoutExerciseEntity(exerciseId = 2, name = "Bench Press (Barbell)", position = 1)
    private val squatSets = listOf(
        ActiveWorkoutSetEntity(exerciseId = 1, position = 0, weightKg = "100", reps = "5", completed = true),
        ActiveWorkoutSetEntity(exerciseId = 1, position = 1, weightKg = "102,5", reps = "3", completed = false)
    )
    private val benchPressSet = ActiveWorkoutSetEntity(exerciseId = 2, position = 0, weightKg = "60", reps = "8", completed = false)

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
        assertEquals(emptyList<ActiveWorkoutExerciseWithSets>(), dao.getWorkout())
    }

    @Test
    fun aSavedWorkoutIsReadBackWithItsExercisesInPosition() = runTest {
        dao.replaceWorkout(listOf(benchPress, squat), squatSets + benchPressSet)

        val workout = dao.getWorkout()

        assertEquals(listOf(squat, benchPress), workout.map { it.exercise })
        assertEquals(squatSets, workout[0].sets.sortedBy { it.position })
        assertEquals(listOf(benchPressSet), workout[1].sets)
    }

    @Test
    fun replacingTheWorkoutLeavesNothingOfTheOldOne() = runTest {
        dao.replaceWorkout(listOf(squat, benchPress), squatSets + benchPressSet)

        dao.replaceWorkout(listOf(squat), listOf(squatSets[0]))

        assertEquals(listOf(ActiveWorkoutExerciseWithSets(squat, listOf(squatSets[0]))), dao.getWorkout())
    }

    @Test
    fun anEmptyWorkoutReplacesTheSavedOne() = runTest {
        dao.replaceWorkout(listOf(squat), squatSets)

        dao.replaceWorkout(emptyList(), emptyList())

        assertEquals(emptyList<ActiveWorkoutExerciseWithSets>(), dao.getWorkout())
    }
}
