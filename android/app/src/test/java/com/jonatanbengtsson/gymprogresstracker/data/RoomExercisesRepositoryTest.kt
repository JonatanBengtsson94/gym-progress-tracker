package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseDao
import com.jonatanbengtsson.gymprogresstracker.data.local.ExerciseEntity
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test

class RoomExercisesRepositoryTest {

    private class FakeExerciseDao : ExerciseDao {
        val exercises = MutableStateFlow(emptyList<ExerciseEntity>())

        override fun observeExercises() = exercises

        override suspend fun replaceExercises(isGlobal: Boolean, exercises: List<ExerciseEntity>) {
            this.exercises.value = this.exercises.value.filter { it.isGlobal != isGlobal } + exercises
        }

        override suspend fun deleteExercises(isGlobal: Boolean) = error("Replaced as a whole by replaceExercises")
        override suspend fun insertExercises(exercises: List<ExerciseEntity>) = error("Replaced as a whole by replaceExercises")
    }

    private class FakeExercisesApi(
        var globalResult: ApiResult<List<Exercise>>,
        var ownResult: ApiResult<List<Exercise>>
    ) : ExercisesApi {
        var globalCalls = 0
        var ownCalls = 0

        override suspend fun getGlobalExercises(): ApiResult<List<Exercise>> {
            globalCalls++
            return globalResult
        }

        override suspend fun getExercises(): ApiResult<List<Exercise>> {
            ownCalls++
            return ownResult
        }
    }

    private val benchPress = Exercise(testId(1), "Bench Press (Barbell)")
    private val squat = Exercise(testId(2), "Squat (Barbell)")
    private val landminePress = Exercise(testId(3), "Landmine Press")

    private val dao = FakeExerciseDao()
    private val api = FakeExercisesApi(ApiResult.Success(listOf(benchPress, squat)), ApiResult.Success(listOf(landminePress)))
    private val repository = RoomExercisesRepository(api, dao)

    @Test
    fun `nothing is stored until the first fetch`() = runTest {
        assertEquals(emptyList<Exercise>(), repository.exercises.first())
    }

    @Test
    fun `a refresh stores the global and the user's own exercises`() = runTest {
        assertEquals(RefreshResult.Success, repository.refresh())

        assertEquals(
            listOf(
                ExerciseEntity(benchPress.id, benchPress.name, isGlobal = true),
                ExerciseEntity(squat.id, squat.name, isGlobal = true),
                ExerciseEntity(landminePress.id, landminePress.name, isGlobal = false)
            ),
            dao.exercises.value
        )
    }

    @Test
    fun `both kinds are listed together, sorted by name ignoring case`() = runTest {
        api.ownResult = ApiResult.Success(listOf(landminePress, Exercise(testId(4), "bench press (barbell)")))

        repository.refresh()

        assertEquals(
            listOf("Bench Press (Barbell)", "bench press (barbell)", "Landmine Press", "Squat (Barbell)"),
            repository.exercises.first().map { it.name }
        )
    }

    @Test
    fun `a refresh replaces what was stored`() = runTest {
        repository.refresh()
        api.globalResult = ApiResult.Success(listOf(squat))
        api.ownResult = ApiResult.Success(emptyList())

        repository.refresh()

        assertEquals(listOf(squat), repository.exercises.first())
    }

    @Test
    fun `without a session the global exercises are still refreshed`() = runTest {
        api.ownResult = ApiResult.Unauthorized

        assertEquals(RefreshResult.NotLoggedIn, repository.refresh())

        assertEquals(listOf(benchPress, squat), repository.exercises.first())
    }

    @Test
    fun `a failed global refresh keeps everything stored and fetches nothing more`() = runTest {
        repository.refresh()

        for ((result, expected) in listOf(
            ApiResult.NetworkError to RefreshResult.NetworkError,
            ApiResult.ServerError to RefreshResult.ServerError
        )) {
            api.globalResult = result

            assertEquals(expected, repository.refresh())
            assertEquals(listOf(benchPress, landminePress, squat), repository.exercises.first())
        }
        assertEquals(1, api.ownCalls)
    }

    @Test
    fun `a failed refresh of the user's own exercises keeps them`() = runTest {
        repository.refresh()
        api.ownResult = ApiResult.NetworkError

        assertEquals(RefreshResult.NetworkError, repository.refresh())

        assertEquals(listOf(benchPress, landminePress, squat), repository.exercises.first())
    }

    @Test
    fun `a fresh install fetches only the global exercises`() = runTest {
        repository.fetchGlobalIfNoneStored()

        assertEquals(1, api.globalCalls)
        assertEquals(0, api.ownCalls)
        assertEquals(listOf(benchPress, squat), repository.exercises.first())
    }

    @Test
    fun `the global exercises aren't fetched again once stored`() = runTest {
        repository.fetchGlobalIfNoneStored()

        repository.fetchGlobalIfNoneStored()

        assertEquals(1, api.globalCalls)
    }

    @Test
    fun `the user's own exercises alone don't stop the global ones being fetched`() = runTest {
        dao.exercises.value = listOf(ExerciseEntity(landminePress.id, landminePress.name, isGlobal = false))

        repository.fetchGlobalIfNoneStored()

        assertEquals(1, api.globalCalls)
    }

    @Test
    fun `a failed fetch of the global exercises is tried again next time`() = runTest {
        api.globalResult = ApiResult.NetworkError
        repository.fetchGlobalIfNoneStored()
        api.globalResult = ApiResult.Success(listOf(benchPress))

        repository.fetchGlobalIfNoneStored()

        assertEquals(2, api.globalCalls)
        assertEquals(listOf(benchPress), repository.exercises.first())
    }
}
