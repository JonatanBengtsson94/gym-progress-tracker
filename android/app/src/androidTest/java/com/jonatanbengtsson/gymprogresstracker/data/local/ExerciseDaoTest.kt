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

    private val squat = ExerciseEntity(exerciseId = testId(1), name = "Squat (Barbell)", isGlobal = true)
    private val benchPress = ExerciseEntity(exerciseId = testId(2), name = "Bench Press (Barbell)", isGlobal = true)
    private val landminePress = ExerciseEntity(exerciseId = testId(3), name = "Landmine Press", isGlobal = false)

    @Before
    fun setUp() {
        database = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), GymDatabase::class.java).build()
        dao = database.exerciseDao()
    }

    @After
    fun tearDown() {
        database.close()
    }

    private suspend fun stored() = dao.observeExercises().first().toSet()

    @Test
    fun aNewDatabaseHasNoExercises() = runTest {
        assertEquals(emptySet<ExerciseEntity>(), stored())
    }

    @Test
    fun storedExercisesAreReadBack() = runTest {
        dao.replaceExercises(isGlobal = true, listOf(squat, benchPress))
        dao.replaceExercises(isGlobal = false, listOf(landminePress))

        assertEquals(setOf(squat, benchPress, landminePress), stored())
    }

    @Test
    fun replacingTheGlobalExercisesKeepsTheUsersOwn() = runTest {
        dao.replaceExercises(isGlobal = true, listOf(squat, benchPress))
        dao.replaceExercises(isGlobal = false, listOf(landminePress))

        dao.replaceExercises(isGlobal = true, listOf(squat))

        assertEquals(setOf(squat, landminePress), stored())
    }

    @Test
    fun replacingTheUsersOwnKeepsTheGlobalExercises() = runTest {
        dao.replaceExercises(isGlobal = true, listOf(squat, benchPress))
        dao.replaceExercises(isGlobal = false, listOf(landminePress))

        dao.replaceExercises(isGlobal = false, emptyList())

        assertEquals(setOf(squat, benchPress), stored())
    }
}
