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
    private val repository = RoomTemplatesRepository(api, dao)

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
            ApiResult.Unauthorized to RefreshResult.SessionExpired
        )) {
            api.result = result

            assertEquals(expected, repository.refresh())
            assertEquals(listOf(pushDay, legDay), repository.templates.first())
        }
    }
}
