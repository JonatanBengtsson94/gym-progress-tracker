package com.jonatanbengtsson.gymprogresstracker.ui.start

import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkout
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.FakeActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeTemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeWorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.LatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.SetEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExercise
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutSet
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.data.testId
import com.jonatanbengtsson.gymprogresstracker.ui.login.MainDispatcherRule
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import java.time.Clock
import java.time.Instant
import java.time.ZoneOffset

class StartWorkoutViewModelTest {

    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val templates = listOf(
        WorkoutTemplate(id = testId(1), name = "Push Day", latestWorkout = null),
        WorkoutTemplate(id = testId(2), name = "Leg Day", latestWorkout = null)
    )

    private val squat = Exercise(testId(1), "Squat (Barbell)")
    private val startedAt = Instant.parse("2026-10-04T17:00:00Z")
    private val workout = ActiveWorkout(startedAt, exercises = listOf(WorkoutExerciseEntry(squat)))
    private val now = Instant.parse("2026-10-04T18:30:00Z")

    private val templatesRepository = FakeTemplatesRepository()
    private val activeWorkoutRepository = FakeActiveWorkoutRepository()
    private val workoutsRepository = FakeWorkoutsRepository()

    // Created lazily so each test can set up the fakes first.
    private val viewModel by lazy {
        StartWorkoutViewModel(templatesRepository, activeWorkoutRepository, workoutsRepository, Clock.fixed(now, ZoneOffset.UTC))
    }

    @Test
    fun `shows the stored templates`() {
        templatesRepository.templates.value = templates.take(1)
        viewModel

        templatesRepository.templates.value = templates

        assertEquals(StartWorkoutUiState(templates = templates), viewModel.uiState)
    }

    @Test
    fun `a workout is in progress once it has an exercise`() {
        assertFalse(viewModel.uiState.workoutInProgress)

        activeWorkoutRepository.update { it.copy(exercises = listOf(WorkoutExerciseEntry(squat))) }

        assertTrue(viewModel.uiState.workoutInProgress)
    }

    @Test
    fun `a workout that hasn't been read yet is not in progress`() {
        activeWorkoutRepository.workout.value = null

        assertFalse(viewModel.uiState.workoutInProgress)
    }

    @Test
    fun `the workout in progress shows its name`() {
        activeWorkoutRepository.workout.value = workout.copy(name = "Push day")

        assertEquals("Push day", viewModel.uiState.workoutName)
    }

    @Test
    fun `starting a new workout starts it now`() {
        viewModel.startNewWorkout()

        assertEquals(ActiveWorkout(startedAt = now), activeWorkoutRepository.workout.value)
        assertEquals(now, viewModel.uiState.workoutStartedAt)
    }

    @Test
    fun `starting a new workout restarts one that has no exercises`() {
        activeWorkoutRepository.workout.value = ActiveWorkout(startedAt)

        viewModel.startNewWorkout()

        assertEquals(ActiveWorkout(startedAt = now), activeWorkoutRepository.workout.value)
    }

    @Test
    fun `starting a new workout keeps one in progress`() {
        activeWorkoutRepository.workout.value = workout

        viewModel.startNewWorkout()

        assertEquals(workout, activeWorkoutRepository.workout.value)
    }

    @Test
    fun `starting from a template fills in its latest workout's sets without completing them`() {
        val bench = Exercise(testId(2), "Bench Press (Barbell)")
        val pullUp = Exercise(testId(3), "Pull Up")
        val template = WorkoutTemplate(
            id = testId(1),
            name = "Push Day",
            latestWorkout = LatestWorkout(
                workoutId = testId(7),
                startedAt = startedAt,
                completedAt = startedAt.plusSeconds(3600),
                exercises = listOf(
                    WorkoutExercise(bench.id, bench.name, listOf(WorkoutSet(8, 60000), WorkoutSet(6, 62500), WorkoutSet(5, 62250))),
                    WorkoutExercise(pullUp.id, pullUp.name, listOf(WorkoutSet(10, 0)))
                )
            )
        )

        viewModel.startFromTemplate(template)

        assertEquals(
            ActiveWorkout(
                startedAt = now,
                name = "Push Day",
                exercises = listOf(
                    WorkoutExerciseEntry(
                        bench,
                        listOf(SetEntry("60", "8", id = 0), SetEntry("62.5", "6", id = 1), SetEntry("62.25", "5", id = 2))
                    ),
                    WorkoutExerciseEntry(pullUp, listOf(SetEntry("", "10", id = 0)))
                )
            ),
            activeWorkoutRepository.workout.value
        )
        assertTrue(viewModel.uiState.workoutInProgress)
    }

    @Test
    fun `starting from a template never performed starts an empty workout named after it`() {
        viewModel.startFromTemplate(templates[1])

        assertEquals(ActiveWorkout(startedAt = now, name = "Leg Day"), activeWorkoutRepository.workout.value)
    }

    @Test
    fun `starting from a template restarts a workout that has no exercises`() {
        activeWorkoutRepository.workout.value = ActiveWorkout(startedAt, name = "Push day")

        viewModel.startFromTemplate(templates[1])

        assertEquals(ActiveWorkout(startedAt = now, name = "Leg Day"), activeWorkoutRepository.workout.value)
    }

    @Test
    fun `starting from a template keeps a workout in progress`() {
        activeWorkoutRepository.workout.value = workout

        viewModel.startFromTemplate(templates[1])

        assertEquals(workout, activeWorkoutRepository.workout.value)
    }

    @Test
    fun `the workout stays in progress when the templates arrive`() {
        activeWorkoutRepository.workout.value = workout
        viewModel

        templatesRepository.templates.value = templates

        assertEquals(
            StartWorkoutUiState(templates = templates, workoutInProgress = true, workoutStartedAt = startedAt),
            viewModel.uiState
        )
    }

    @Test
    fun `discarding the workout empties it and clears when it started`() {
        activeWorkoutRepository.workout.value = workout

        viewModel.discardWorkout()

        assertEquals(ActiveWorkout(), activeWorkoutRepository.workout.value)
        assertFalse(viewModel.uiState.workoutInProgress)
        assertNull(viewModel.uiState.workoutStartedAt)
    }

    @Test
    fun `shows how many saved workouts are waiting to sync`() {
        viewModel

        workoutsRepository.pendingCount.value = 2

        assertEquals(2, viewModel.uiState.pendingWorkouts)
    }
}
