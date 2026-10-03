package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseDao
import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseEntity
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test

class RoomExercisesRepositoryTest {

    private class FakeExerciseDao : ExerciseDao {
        val exercises = MutableStateFlow(emptyList<ExerciseEntity>())

        override fun observeExercises() = exercises.map { stored -> stored.sortedBy { it.position } }

        override suspend fun replaceExercises(exercises: List<ExerciseEntity>) {
            this.exercises.value = exercises
        }

        override suspend fun deleteExercises() = error("Replaced as a whole by replaceExercises")
        override suspend fun insertExercises(exercises: List<ExerciseEntity>) = error("Replaced as a whole by replaceExercises")
    }

    private class FakeExercisesApi(var result: ExercisesResult) : ExercisesApi {
        val calls = mutableListOf<String>()

        override suspend fun getExercises(sessionId: String): ExercisesResult {
            calls += sessionId
            return result
        }
    }

    private val benchPress = Exercise(testId(1), "Bench Press (Barbell)")
    private val squat = Exercise(testId(2), "Squat (Barbell)")

    private val dao = FakeExerciseDao()
    private val api = FakeExercisesApi(ExercisesResult.Success(listOf(squat, benchPress)))
    private val repository = RoomExercisesRepository(api, dao)

    @Test
    fun `nothing is stored until the first refresh`() = runTest {
        assertEquals(emptyList<Exercise>(), repository.exercises.first())
    }

    @Test
    fun `a refresh stores the server's exercises in its order`() = runTest {
        assertEquals(RefreshResult.Success, repository.refresh("session-123"))

        assertEquals(listOf("session-123"), api.calls)
        assertEquals(listOf(ExerciseEntity(testId(2), "Squat (Barbell)", 0), ExerciseEntity(testId(1), "Bench Press (Barbell)", 1)), dao.exercises.value)
        assertEquals(listOf(squat, benchPress), repository.exercises.first())
    }

    @Test
    fun `a refresh replaces what was stored`() = runTest {
        repository.refresh("session-123")
        api.result = ExercisesResult.Success(listOf(benchPress))

        repository.refresh("session-123")

        assertEquals(listOf(benchPress), repository.exercises.first())
    }

    @Test
    fun `a failed refresh keeps the stored exercises and says why`() = runTest {
        repository.refresh("session-123")

        for ((result, expected) in listOf(
            ExercisesResult.NetworkError to RefreshResult.NetworkError,
            ExercisesResult.ServerError to RefreshResult.ServerError,
            ExercisesResult.SessionExpired to RefreshResult.SessionExpired
        )) {
            api.result = result

            assertEquals(expected, repository.refresh("session-123"))
            assertEquals(listOf(squat, benchPress), repository.exercises.first())
        }
    }
}
