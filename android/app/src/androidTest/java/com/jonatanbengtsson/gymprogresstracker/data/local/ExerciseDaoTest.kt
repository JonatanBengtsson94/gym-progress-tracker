package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.jonatanbengtsson.gymprogresstracker.data.testId
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class ExerciseDaoTest {

    private lateinit var database: GymDatabase
    private lateinit var dao: ExerciseDao

    private val squat = ExerciseEntity(exerciseId = testId(1), name = "Squat (Barbell)", position = 1)
    private val benchPress = ExerciseEntity(exerciseId = testId(2), name = "Bench Press (Barbell)", position = 0)

    @Before
    fun setUp() {
        database = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), GymDatabase::class.java).build()
        dao = database.exerciseDao()
    }

    @After
    fun tearDown() {
        database.close()
    }

    @Test
    fun aNewDatabaseHasNoExercises() = runTest {
        assertEquals(emptyList<ExerciseEntity>(), dao.observeExercises().first())
    }

    @Test
    fun storedExercisesAreReadBackInPosition() = runTest {
        dao.replaceExercises(listOf(squat, benchPress))

        assertEquals(listOf(benchPress, squat), dao.observeExercises().first())
    }

    @Test
    fun replacingTheExercisesLeavesNothingOfTheOldOnes() = runTest {
        dao.replaceExercises(listOf(squat, benchPress))

        dao.replaceExercises(listOf(squat.copy(position = 0)))

        assertEquals(listOf(squat.copy(position = 0)), dao.observeExercises().first())
    }
}
