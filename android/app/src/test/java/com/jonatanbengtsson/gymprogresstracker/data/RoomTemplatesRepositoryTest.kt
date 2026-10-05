package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateDao
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateLatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateSetEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateWithSets
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.Instant
import kotlin.uuid.Uuid

class RoomTemplatesRepositoryTest {

    /** Stores the templates in memory, handing sets back in reverse since Room doesn't promise their order. */
    private class FakeTemplateDao : TemplateDao {
        val templates = MutableStateFlow(emptyList<TemplateEntity>())
        val sets = MutableStateFlow(emptyList<TemplateSetEntity>())

        override fun observeTemplates() = combine(templates, sets) { templates, sets ->
            templates.sortedBy { it.position }.map { template ->
                TemplateWithSets(template, sets.filter { it.templateId == template.templateId }.reversed())
            }
        }

        override suspend fun replaceTemplates(templates: List<TemplateEntity>, sets: List<TemplateSetEntity>) {
            this.templates.value = templates
            this.sets.value = sets
        }

        override suspend fun deleteTemplates() = error("Replaced as a whole by replaceTemplates")
        override suspend fun insertTemplates(templates: List<TemplateEntity>) = error("Replaced as a whole by replaceTemplates")
        override suspend fun insertSets(sets: List<TemplateSetEntity>) = error("Replaced as a whole by replaceTemplates")
    }

    private class FakeTemplatesApi(var result: ApiResult<List<WorkoutTemplate>>) : TemplatesApi {
        var calls = 0

        override suspend fun getTemplates(): ApiResult<List<WorkoutTemplate>> {
            calls++
            return result
        }

        override suspend fun createTemplate(templateId: Uuid, name: String) = error("Not used by the templates repository")
    }

    private val pushDay = WorkoutTemplate(
        id = testId(10),
        name = "Push Day",
        latestWorkout = LatestWorkout(
            workoutId = testId(20),
            startedAt = Instant.parse("2026-09-30T17:00:00Z"),
            completedAt = Instant.parse("2026-09-30T18:00:00Z"),
            exercises = listOf(
                WorkoutExercise(testId(1), "Bench Press (Barbell)", listOf(WorkoutSet(8, 60000), WorkoutSet(6, 62500))),
                WorkoutExercise(testId(2), "Overhead Press (Barbell)", listOf(WorkoutSet(10, 30000)))
            )
        )
    )
    private val legDay = WorkoutTemplate(id = testId(11), name = "Leg Day", latestWorkout = null)

    private val dao = FakeTemplateDao()
    private val api = FakeTemplatesApi(ApiResult.Success(listOf(pushDay, legDay)))
    private val pendingWorkoutsRepository = FakePendingWorkoutsRepository()
    private val repository = RoomTemplatesRepository(api, dao, pendingWorkoutsRepository)

    private val squat = WorkoutExercise(testId(3), "Squat (Barbell)", listOf(WorkoutSet(5, 100000)))

    /** A workout completed at [completedAt] waiting to sync, logged under the template [templateId] named [name]. */
    private fun pendingWorkout(
        workoutId: Uuid,
        templateId: Uuid,
        name: String,
        completedAt: Instant,
        templateIsNew: Boolean = false
    ) = PendingWorkout(
        workoutId,
        templateId,
        templateIsNew,
        FinishedWorkout(name, completedAt.minusSeconds(3600), completedAt, listOf(squat))
    )

    private fun PendingWorkout.asLatest() =
        LatestWorkout(workoutId, workout.startedAt, workout.completedAt, workout.exercises)

    @Test
    fun `nothing is stored until the first refresh`() = runTest {
        assertEquals(emptyList<WorkoutTemplate>(), repository.templates.first())
    }

    @Test
    fun `refreshed templates are read back as the server sent them`() = runTest {
        assertEquals(RefreshResult.Success, repository.refresh())

        assertEquals(1, api.calls)
        assertEquals(listOf(pushDay, legDay), repository.templates.first())
    }

    @Test
    fun `a template's latest workout is stored with its sets in the order they were logged`() = runTest {
        repository.refresh()

        val latest = TemplateLatestWorkout(testId(20), Instant.parse("2026-09-30T17:00:00Z"), Instant.parse("2026-09-30T18:00:00Z"))
        assertEquals(
            listOf(TemplateEntity(testId(10), "Push Day", 0, latest), TemplateEntity(testId(11), "Leg Day", 1, null)),
            dao.templates.value
        )
        assertEquals(
            listOf(
                TemplateSetEntity(testId(10), 0, testId(1), "Bench Press (Barbell)", 8, 60000),
                TemplateSetEntity(testId(10), 1, testId(1), "Bench Press (Barbell)", 6, 62500),
                TemplateSetEntity(testId(10), 2, testId(2), "Overhead Press (Barbell)", 10, 30000)
            ),
            dao.sets.value
        )
    }

    @Test
    fun `a refresh replaces what was stored`() = runTest {
        repository.refresh()
        api.result = ApiResult.Success(listOf(legDay))

        repository.refresh()

        assertEquals(listOf(legDay), repository.templates.first())
        assertEquals(emptyList<TemplateSetEntity>(), dao.sets.value)
    }

    @Test
    fun `a failed refresh keeps the stored templates and says why`() = runTest {
        repository.refresh()

        for ((result, expected) in listOf(
            ApiResult.NetworkError to RefreshResult.NetworkError,
            ApiResult.ServerError to RefreshResult.ServerError,
            ApiResult.Unauthorized to RefreshResult.NotLoggedIn
        )) {
            api.result = result

            assertEquals(expected, repository.refresh())
            assertEquals(listOf(pushDay, legDay), repository.templates.first())
        }
    }

    @Test
    fun `a workout waiting to sync under a new template shows as that template, most recent first`() = runTest {
        repository.refresh()
        val chestDay = pendingWorkout(testId(30), testId(12), "Chest Day", Instant.parse("2026-10-04T18:00:00Z"), templateIsNew = true)

        pendingWorkoutsRepository.add(chestDay)

        assertEquals(
            listOf(WorkoutTemplate(testId(12), "Chest Day", chestDay.asLatest()), pushDay, legDay),
            repository.templates.first()
        )
    }

    @Test
    fun `workouts waiting to sync show before the templates have ever been fetched`() = runTest {
        val chestDay = pendingWorkout(testId(30), testId(12), "Chest Day", Instant.parse("2026-10-04T18:00:00Z"), templateIsNew = true)

        pendingWorkoutsRepository.add(chestDay)

        assertEquals(listOf(WorkoutTemplate(testId(12), "Chest Day", chestDay.asLatest())), repository.templates.first())
    }

    @Test
    fun `workouts waiting to sync under the same new template show it once, with the newest as its latest`() = runTest {
        val first = pendingWorkout(testId(30), testId(12), "Chest Day", Instant.parse("2026-10-03T18:00:00Z"), templateIsNew = true)
        val second = pendingWorkout(testId(31), testId(12), "Chest Day", Instant.parse("2026-10-04T18:00:00Z"), templateIsNew = true)

        pendingWorkoutsRepository.add(first)
        pendingWorkoutsRepository.add(second)

        assertEquals(listOf(WorkoutTemplate(testId(12), "Chest Day", second.asLatest())), repository.templates.first())
    }

    @Test
    fun `a newer workout waiting to sync becomes its template's latest and moves it first`() = runTest {
        repository.refresh()
        val legs = pendingWorkout(testId(30), legDay.id, "Leg Day", Instant.parse("2026-10-04T18:00:00Z"))

        pendingWorkoutsRepository.add(legs)

        assertEquals(listOf(legDay.copy(latestWorkout = legs.asLatest()), pushDay), repository.templates.first())
    }

    @Test
    fun `an older workout waiting to sync doesn't replace its template's latest`() = runTest {
        repository.refresh()

        pendingWorkoutsRepository.add(pendingWorkout(testId(30), pushDay.id, "Push Day", Instant.parse("2026-09-01T18:00:00Z")))

        assertEquals(listOf(pushDay, legDay), repository.templates.first())
    }

    @Test
    fun `a new template with the name of one already fetched shows under that one`() = runTest {
        repository.refresh()
        val legs = pendingWorkout(testId(30), testId(12), "leg day", Instant.parse("2026-10-04T18:00:00Z"), templateIsNew = true)

        pendingWorkoutsRepository.add(legs)

        assertEquals(listOf(legDay.copy(latestWorkout = legs.asLatest()), pushDay), repository.templates.first())
    }

    @Test
    fun `a workout saved with a new name shows in the templates until and after it's synced`() = runTest {
        // The real repositories over one queue, as in the app.
        val chestDayId = testId(12)
        val templatesApi = object : TemplatesApi {
            var templates = listOf(pushDay, legDay)
            override suspend fun getTemplates() = ApiResult.Success(templates)
            override suspend fun createTemplate(templateId: Uuid, name: String) = ApiResult.Success(chestDayId)
        }
        val workoutsApi = object : WorkoutsApi {
            override suspend fun putWorkout(workoutId: Uuid, templateId: Uuid, workout: FinishedWorkout): ApiResult<Unit> {
                val latest = LatestWorkout(workoutId, workout.startedAt, workout.completedAt, workout.exercises)
                templatesApi.templates = listOf(WorkoutTemplate(templateId, workout.name, latest)) + templatesApi.templates
                return ApiResult.Success(Unit)
            }
        }
        val templatesRepository = RoomTemplatesRepository(templatesApi, dao, pendingWorkoutsRepository)
        val workoutsRepository = ApiWorkoutsRepository(workoutsApi, templatesApi, pendingWorkoutsRepository, templatesRepository)
        val exercisesRepository = FakeExercisesRepository().apply { refreshResult.complete(RefreshResult.Success) }
        val syncRepository = DefaultSyncRepository(workoutsRepository, templatesRepository, exercisesRepository)
        templatesRepository.refresh()
        val chestDay = FinishedWorkout("Chest Day", Instant.parse("2026-10-04T17:00:00Z"), Instant.parse("2026-10-04T18:00:00Z"), listOf(squat))

        workoutsRepository.save(testId(30), chestDay)

        val beforeSync = templatesRepository.templates.first()
        assertEquals(listOf("Chest Day", "Push Day", "Leg Day"), beforeSync.map { it.name })
        assertEquals(LatestWorkout(testId(30), chestDay.startedAt, chestDay.completedAt, chestDay.exercises), beforeSync[0].latestWorkout)

        syncRepository.sync()

        val afterSync = templatesRepository.templates.first()
        assertEquals(listOf(chestDayId, pushDay.id, legDay.id), afterSync.map { it.id })
        assertEquals(beforeSync[0].latestWorkout, afterSync[0].latestWorkout)
    }
}
