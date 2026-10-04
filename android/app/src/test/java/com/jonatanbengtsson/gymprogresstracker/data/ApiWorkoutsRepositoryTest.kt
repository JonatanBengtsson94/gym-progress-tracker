package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.Instant
import kotlin.uuid.Uuid

@OptIn(ExperimentalCoroutinesApi::class)
class ApiWorkoutsRepositoryTest {

    private class FakeWorkoutsApi : WorkoutsApi {
        var result: ApiResult<Unit> = ApiResult.Success(Unit)
        val puts = mutableListOf<Pair<Uuid, FinishedWorkout>>()

        override suspend fun putWorkout(workoutId: Uuid, workout: FinishedWorkout): ApiResult<Unit> {
            puts += workoutId to workout
            return result
        }
    }

    private val api = FakeWorkoutsApi()
    private val pushDay = WorkoutTemplate(testId(10), "Push Day", latestWorkout = null)
    private val templatesRepository = FakeTemplatesRepository(listOf(pushDay))

    private val workout = FinishedWorkout(
        templateId = null,
        templateName = "Push Day",
        startedAt = Instant.parse("2026-10-04T17:00:00Z"),
        completedAt = Instant.parse("2026-10-04T18:00:00Z"),
        exercises = listOf(FinishedExercise(testId(1), listOf(WorkoutSet(5, 100000))))
    )

    private fun TestScope.repository() = ApiWorkoutsRepository(api, templatesRepository, backgroundScope)

    @Test
    fun `a workout named like a template is saved under it, ignoring case`() = runTest {
        repository().save(testId(20), workout.copy(templateName = " push day "))

        assertEquals(testId(20) to workout.copy(templateName = " push day ", templateId = pushDay.id), api.puts.single())
    }

    @Test
    fun `a workout with a new name is saved under a new template`() = runTest {
        repository().save(testId(20), workout.copy(templateName = "Leg Day"))

        assertEquals(null, api.puts.single().second.templateId)
    }

    @Test
    fun `a workout's own template id is kept`() = runTest {
        repository().save(testId(20), workout.copy(templateId = testId(11)))

        assertEquals(testId(11), api.puts.single().second.templateId)
    }

    @Test
    fun `a saved workout refreshes the templates`() = runTest {
        assertEquals(ApiResult.Success(Unit), repository().save(testId(20), workout))
        runCurrent()

        assertEquals(1, templatesRepository.refreshes)
    }

    @Test
    fun `a failed save leaves the templates alone`() = runTest {
        api.result = ApiResult.NetworkError

        assertEquals(ApiResult.NetworkError, repository().save(testId(20), workout))
        runCurrent()

        assertEquals(0, templatesRepository.refreshes)
    }
}
