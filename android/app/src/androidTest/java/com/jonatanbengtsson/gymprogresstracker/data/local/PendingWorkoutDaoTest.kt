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
import java.time.Instant

@RunWith(AndroidJUnit4::class)
class PendingWorkoutDaoTest {

    private lateinit var database: GymDatabase
    private lateinit var dao: PendingWorkoutDao

    private val pushDay = PendingWorkoutEntity(
        workoutId = testId(20),
        templateId = testId(10),
        templateIsNew = true,
        name = "Push day",
        startedAt = Instant.parse("2026-10-04T17:00:00Z"),
        completedAt = Instant.parse("2026-10-04T18:00:00Z")
    )
    private val legDay = PendingWorkoutEntity(
        workoutId = testId(21),
        templateId = testId(11),
        templateIsNew = false,
        name = "Leg day",
        startedAt = Instant.parse("2026-10-03T17:00:00Z"),
        completedAt = Instant.parse("2026-10-03T18:00:00Z")
    )
    private val pushDaySets = listOf(
        PendingWorkoutSetEntity(testId(20), position = 0, exerciseId = testId(1), exerciseName = "Squat (Barbell)", reps = 5, weightGrams = 100000),
        PendingWorkoutSetEntity(testId(20), position = 1, exerciseId = testId(2), exerciseName = "Bench Press (Barbell)", reps = 8, weightGrams = 60000)
    )
    private val legDaySet =
        PendingWorkoutSetEntity(testId(21), position = 0, exerciseId = testId(3), exerciseName = "Deadlift", reps = 3, weightGrams = 140000)

    @Before
    fun setUp() {
        database = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), GymDatabase::class.java).build()
        dao = database.pendingWorkoutDao()
    }

    @After
    fun tearDown() {
        database.close()
    }

    @Test
    fun storedWorkoutsAreReadBackOldestFirstWithTheirSets() = runTest {
        dao.replaceWorkout(pushDay, pushDaySets)
        dao.replaceWorkout(legDay, listOf(legDaySet))

        val workouts = dao.observeWorkouts().first()

        assertEquals(listOf(legDay, pushDay), workouts.map { it.workout })
        assertEquals(listOf(legDaySet), workouts[0].sets)
        assertEquals(pushDaySets, workouts[1].sets.sortedBy { it.position })
    }

    @Test
    fun storingAWorkoutAgainReplacesItAndItsSets() = runTest {
        dao.replaceWorkout(pushDay, pushDaySets)

        val renamed = pushDay.copy(name = "Chest day")
        dao.replaceWorkout(renamed, pushDaySets.take(1))

        val workout = dao.observeWorkouts().first().single()
        assertEquals(renamed, workout.workout)
        assertEquals(pushDaySets.take(1), workout.sets)
    }

    @Test
    fun deletingAWorkoutDeletesItsSetsAndKeepsTheOthers() = runTest {
        dao.replaceWorkout(pushDay, pushDaySets)
        dao.replaceWorkout(legDay, listOf(legDaySet))

        dao.deleteWorkout(pushDay.workoutId)
        // Inserted bare, so any of its sets left behind would show up again.
        dao.insertWorkout(pushDay)

        val workouts = dao.observeWorkouts().first()
        assertEquals(listOf(legDay, pushDay), workouts.map { it.workout })
        assertEquals(listOf(listOf(legDaySet), emptyList()), workouts.map { it.sets })
    }

    @Test
    fun deletingAllWorkoutsLeavesNone() = runTest {
        dao.replaceWorkout(pushDay, pushDaySets)
        dao.replaceWorkout(legDay, listOf(legDaySet))

        dao.deleteWorkouts()

        assertEquals(emptyList<PendingWorkoutWithSets>(), dao.observeWorkouts().first())
    }
}
