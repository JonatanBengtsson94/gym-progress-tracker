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
class TemplateDaoTest {

    private lateinit var database: GymDatabase
    private lateinit var dao: TemplateDao

    private val pushDay = TemplateEntity(
        templateId = testId(10),
        name = "Push Day",
        position = 0,
        latestWorkout = TemplateLatestWorkout(testId(20), Instant.parse("2026-09-30T17:00:00Z"), Instant.parse("2026-09-30T18:00:00Z"))
    )
    private val legDay = TemplateEntity(templateId = testId(11), name = "Leg Day", position = 1, latestWorkout = null)
    private val pushDaySets = listOf(
        TemplateSetEntity(testId(10), position = 0, exerciseId = testId(1), exerciseName = "Bench Press (Barbell)", reps = 8, weightGrams = 60000),
        TemplateSetEntity(testId(10), position = 1, exerciseId = testId(1), exerciseName = "Bench Press (Barbell)", reps = 6, weightGrams = 62500)
    )

    @Before
    fun setUp() {
        database = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), GymDatabase::class.java).build()
        dao = database.templateDao()
    }

    @After
    fun tearDown() {
        database.close()
    }

    @Test
    fun aNewDatabaseHasNoTemplates() = runTest {
        assertEquals(emptyList<TemplateWithSets>(), dao.observeTemplates().first())
    }

    @Test
    fun storedTemplatesAreReadBackInPositionWithTheirSets() = runTest {
        dao.replaceTemplates(listOf(legDay, pushDay), pushDaySets)

        val templates = dao.observeTemplates().first()

        assertEquals(listOf(pushDay, legDay), templates.map { it.template })
        assertEquals(pushDaySets, templates[0].sets.sortedBy { it.position })
        assertEquals(emptyList<TemplateSetEntity>(), templates[1].sets)
    }

    @Test
    fun aTemplateWithoutALatestWorkoutIsReadBackWithout() = runTest {
        dao.replaceTemplates(listOf(legDay), emptyList())

        assertEquals(null, dao.observeTemplates().first().single().template.latestWorkout)
    }

    @Test
    fun replacingTheTemplatesLeavesNothingOfTheOldOnes() = runTest {
        dao.replaceTemplates(listOf(pushDay, legDay), pushDaySets)

        dao.replaceTemplates(listOf(legDay.copy(position = 0)), emptyList())

        assertEquals(listOf(TemplateWithSets(legDay.copy(position = 0), emptyList())), dao.observeTemplates().first())
    }
}
