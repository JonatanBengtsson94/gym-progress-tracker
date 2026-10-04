package com.jonatanbengtsson.gymprogresstracker.ui.start

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkout
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.FakeActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeSessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeTemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.RefreshResult
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.data.testId
import com.jonatanbengtsson.gymprogresstracker.ui.login.MainDispatcherRule
import kotlinx.coroutines.CompletableDeferred
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
    private val sessionRepository = FakeSessionRepository(SessionState.LoggedIn("session-123"))

    // Created lazily so each test can set up the fakes before the view model refreshes on init.
    private val viewModel by lazy {
        StartWorkoutViewModel(templatesRepository, activeWorkoutRepository, sessionRepository, Clock.fixed(now, ZoneOffset.UTC))
    }

    @Test
    fun `refreshes the templates on creation`() {
        viewModel

        assertEquals(1, templatesRepository.refreshes)
        assertEquals(StartWorkoutUiState(isLoading = true), viewModel.uiState)
    }

    @Test
    fun `stored templates show while they're being refreshed`() {
        templatesRepository.templates.value = templates

        assertEquals(StartWorkoutUiState(isLoading = true, templates = templates), viewModel.uiState)
    }

    @Test
    fun `refreshed templates replace the stored ones`() {
        templatesRepository.templates.value = templates.take(1)
        viewModel

        templatesRepository.templates.value = templates
        templatesRepository.refreshResult.complete(RefreshResult.Success)

        assertEquals(StartWorkoutUiState(templates = templates), viewModel.uiState)
    }

    @Test
    fun `network error shows network error`() {
        assertErrorFor(RefreshResult.NetworkError, R.string.start_workout_error_network)
    }

    @Test
    fun `server error shows server error`() {
        assertErrorFor(RefreshResult.ServerError, R.string.start_workout_error_server)
    }

    @Test
    fun `a failed refresh keeps showing the stored templates`() {
        templatesRepository.templates.value = templates
        viewModel

        templatesRepository.refreshResult.complete(RefreshResult.NetworkError)

        assertEquals(StartWorkoutUiState(templates = templates, errorMessage = R.string.start_workout_error_network), viewModel.uiState)
    }

    @Test
    fun `an expired session shows no error`() {
        viewModel
        templatesRepository.refreshResult.complete(RefreshResult.SessionExpired)

        assertEquals(StartWorkoutUiState(), viewModel.uiState)
    }

    @Test
    fun `logging out ends the session but keeps the workout`() {
        activeWorkoutRepository.workout.value = workout

        viewModel.logOut()

        assertEquals(SessionState.LoggedOut, sessionRepository.session.value)
        assertEquals(workout, activeWorkoutRepository.workout.value)
    }

    @Test
    fun `reload is ignored while a refresh is in flight`() {
        viewModel.loadTemplates()

        assertEquals(1, templatesRepository.refreshes)
    }

    @Test
    fun `retrying after an error clears the error and refreshes again`() {
        viewModel
        templatesRepository.refreshResult.complete(RefreshResult.NetworkError)
        templatesRepository.refreshResult = CompletableDeferred()

        viewModel.loadTemplates()

        assertEquals(StartWorkoutUiState(isLoading = true), viewModel.uiState)
        templatesRepository.templates.value = templates
        templatesRepository.refreshResult.complete(RefreshResult.Success)
        assertEquals(StartWorkoutUiState(templates = templates), viewModel.uiState)
        assertEquals(2, templatesRepository.refreshes)
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
    fun `the workout stays in progress when the templates arrive`() {
        activeWorkoutRepository.workout.value = workout
        viewModel

        templatesRepository.templates.value = templates
        templatesRepository.refreshResult.complete(RefreshResult.Success)

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

    private fun assertErrorFor(result: RefreshResult, expectedMessage: Int) {
        viewModel
        templatesRepository.refreshResult.complete(result)

        assertEquals(StartWorkoutUiState(errorMessage = expectedMessage), viewModel.uiState)
    }
}
