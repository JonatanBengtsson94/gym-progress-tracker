package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test
import java.time.Instant
import kotlin.uuid.Uuid

class ApiWorkoutsRepositoryTest {

    /** Answers each workout with its entry in [results], or success when it has none. */
    private class FakeWorkoutsApi : WorkoutsApi {
        val results = mutableMapOf<Uuid, ApiResult<Unit>>()
        val puts = mutableListOf<Pair<Uuid, Uuid>>()

        override suspend fun putWorkout(workoutId: Uuid, templateId: Uuid, workout: FinishedWorkout): ApiResult<Unit> {
            puts += workoutId to templateId
            return results[workoutId] ?: ApiResult.Success(Unit)
        }
    }

    /** Creates each template under the id in [existing] for its name, or under its own id when there's none. */
    private class FakeTemplatesApi : TemplatesApi {
        val existing = mutableMapOf<String, Uuid>()
        var result: ApiResult<Uuid>? = null
        val creates = mutableListOf<Pair<Uuid, String>>()

        override suspend fun getTemplates() = error("Not used when syncing")

        override suspend fun createTemplate(templateId: Uuid, name: String): ApiResult<Uuid> {
            creates += templateId to name
            return result ?: ApiResult.Success(existing.getOrPut(name.lowercase()) { templateId })
        }
    }

    private val workoutsApi = FakeWorkoutsApi()
    private val templatesApi = FakeTemplatesApi()
    private val pushDay = WorkoutTemplate(testId(10), "Push Day", latestWorkout = null)
    private val templatesRepository = FakeTemplatesRepository(listOf(pushDay))
    private val pendingWorkoutsRepository = FakePendingWorkoutsRepository()
    private val repository = ApiWorkoutsRepository(workoutsApi, templatesApi, pendingWorkoutsRepository, templatesRepository)

    private val workout = FinishedWorkout(
        name = "Push Day",
        startedAt = Instant.parse("2026-10-04T17:00:00Z"),
        completedAt = Instant.parse("2026-10-04T18:00:00Z"),
        exercises = listOf(WorkoutExercise(testId(1), "Squat (Barbell)", listOf(WorkoutSet(5, 100000))))
    )
    private val legDay = workout.copy(name = "Leg Day")

    private fun pending() = pendingWorkoutsRepository.workouts.value

    @Test
    fun `saving keeps the workout on the device without sending it`() = runTest {
        repository.save(testId(20), workout)

        assertEquals(listOf(testId(20)), pending().map { it.workoutId })
        assertEquals(1, repository.pendingCount.first())
        assertEquals(emptyList<Pair<Uuid, Uuid>>(), workoutsApi.puts)
    }

    @Test
    fun `a workout named like a template is saved under it, ignoring case`() = runTest {
        repository.save(testId(20), workout.copy(name = "push day"))

        assertEquals(PendingWorkout(testId(20), pushDay.id, templateIsNew = false, workout.copy(name = "push day")), pending().single())
    }

    @Test
    fun `a workout with a new name is saved under a new template`() = runTest {
        repository.save(testId(20), legDay)

        val saved = pending().single()
        assertEquals(true, saved.templateIsNew)
        assertNotEquals(pushDay.id, saved.templateId)
    }

    @Test
    fun `a workout named like a template another saved workout created shares it, and it's still new`() = runTest {
        repository.save(testId(20), legDay)
        val created = pending().single().templateId
        // As the real repository does, the template created on the device is listed until it's synced.
        templatesRepository.templates.value = listOf(pushDay, WorkoutTemplate(created, "Leg Day", latestWorkout = null))

        repository.save(testId(21), legDay.copy(name = "LEG DAY"))

        assertEquals(PendingWorkout(testId(21), created, templateIsNew = true, legDay.copy(name = "LEG DAY")), pending()[1])
    }

    @Test
    fun `saving a workout again replaces it`() = runTest {
        repository.save(testId(20), workout)

        repository.save(testId(20), workout.copy(startedAt = Instant.parse("2026-10-04T17:30:00Z")))

        assertEquals(listOf(workout.copy(startedAt = Instant.parse("2026-10-04T17:30:00Z"))), pending().map { it.workout })
    }

    @Test
    fun `sending sends the saved workouts in order under their templates and removes them`() = runTest {
        repository.save(testId(20), workout)
        repository.save(testId(21), workout)

        assertEquals(SyncResult.Success, repository.send())

        assertEquals(listOf(testId(20) to pushDay.id, testId(21) to pushDay.id), workoutsApi.puts)
        assertEquals(emptyList<Pair<Uuid, String>>(), templatesApi.creates)
        assertEquals(emptyList<PendingWorkout>(), pending())
    }

    @Test
    fun `a new template is created before its workout is sent`() = runTest {
        repository.save(testId(20), legDay)
        val created = pending().single().templateId

        repository.send()

        assertEquals(listOf(created to "Leg Day"), templatesApi.creates)
        assertEquals(listOf(testId(20) to created), workoutsApi.puts)
    }

    @Test
    fun `a workout goes under the template the server already has with its name`() = runTest {
        templatesApi.existing["leg day"] = testId(12)
        repository.save(testId(20), legDay)

        repository.send()

        assertEquals(listOf(testId(20) to testId(12)), workoutsApi.puts)
    }

    @Test
    fun `a template that couldn't be created keeps its workout and the others are still sent`() = runTest {
        repository.save(testId(20), legDay)
        repository.save(testId(21), workout)
        templatesApi.result = ApiResult.ServerError

        assertEquals(SyncResult.ServerError, repository.send())

        assertEquals(listOf(testId(21) to pushDay.id), workoutsApi.puts)
        assertEquals(listOf(testId(20)), pending().map { it.workoutId })
    }

    @Test
    fun `a rejected workout is kept and the others are still sent`() = runTest {
        repository.save(testId(20), workout)
        repository.save(testId(21), workout)
        workoutsApi.results[testId(20)] = ApiResult.ServerError

        assertEquals(SyncResult.ServerError, repository.send())

        assertEquals(listOf(testId(20), testId(21)), workoutsApi.puts.map { it.first })
        assertEquals(listOf(testId(20)), pending().map { it.workoutId })
    }

    @Test
    fun `a network error stops sending and keeps the rest`() = runTest {
        repository.save(testId(20), workout)
        repository.save(testId(21), workout)
        workoutsApi.results[testId(20)] = ApiResult.NetworkError

        assertEquals(SyncResult.NetworkError, repository.send())

        assertEquals(listOf(testId(20)), workoutsApi.puts.map { it.first })
        assertEquals(listOf(testId(20), testId(21)), pending().map { it.workoutId })
    }

    @Test
    fun `a network error creating a template stops sending`() = runTest {
        repository.save(testId(20), legDay)
        repository.save(testId(21), workout)
        templatesApi.result = ApiResult.NetworkError

        assertEquals(SyncResult.NetworkError, repository.send())

        assertEquals(emptyList<Pair<Uuid, Uuid>>(), workoutsApi.puts)
        assertEquals(2, pending().size)
    }

    @Test
    fun `an expired session stops sending and keeps the workouts`() = runTest {
        repository.save(testId(20), workout)
        workoutsApi.results[testId(20)] = ApiResult.Unauthorized

        assertEquals(SyncResult.NotLoggedIn, repository.send())

        assertEquals(listOf(testId(20)), pending().map { it.workoutId })
    }

    @Test
    fun `sending nothing sends nothing`() = runTest {
        assertEquals(SyncResult.Success, repository.send())

        assertEquals(emptyList<Pair<Uuid, Uuid>>(), workoutsApi.puts)
    }
}
