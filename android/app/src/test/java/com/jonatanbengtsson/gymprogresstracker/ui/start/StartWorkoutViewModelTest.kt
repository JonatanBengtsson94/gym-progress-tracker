package com.jonatanbengtsson.gymprogresstracker.ui.start

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.FakeActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesResult
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.ui.login.MainDispatcherRule
import kotlinx.coroutines.CompletableDeferred
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class StartWorkoutViewModelTest {

    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    /** Suspends each request until the test completes [response]. */
    private class FakeTemplatesApi : TemplatesApi {
        var response = CompletableDeferred<TemplatesResult>()
        val calls = mutableListOf<String>()

        override suspend fun getTemplates(sessionId: String): TemplatesResult {
            calls += sessionId
            return response.await()
        }
    }

    private val templates = listOf(
        WorkoutTemplate(id = 1, name = "Push Day", latestWorkout = null),
        WorkoutTemplate(id = 2, name = "Leg Day", latestWorkout = null)
    )

    private val squat = Exercise(1, "Squat (Barbell)")

    private val templatesApi = FakeTemplatesApi()
    private val activeWorkoutRepository = FakeActiveWorkoutRepository()

    // Created lazily so each test can set up the fakes before the view model loads on init.
    private val viewModel by lazy { StartWorkoutViewModel(templatesApi, activeWorkoutRepository, "session-123") }

    @Test
    fun `loads templates with the session id on creation`() {
        viewModel

        assertEquals(listOf("session-123"), templatesApi.calls)
        assertEquals(StartWorkoutUiState(isLoading = true), viewModel.uiState)
    }

    @Test
    fun `success exposes the templates in order`() {
        viewModel
        templatesApi.response.complete(TemplatesResult.Success(templates))

        assertEquals(StartWorkoutUiState(templates = templates), viewModel.uiState)
    }

    @Test
    fun `network error shows network error`() {
        assertErrorFor(TemplatesResult.NetworkError, R.string.start_workout_error_network)
    }

    @Test
    fun `server error shows server error`() {
        assertErrorFor(TemplatesResult.ServerError, R.string.start_workout_error_server)
    }

    @Test
    fun `expired session is reported`() {
        viewModel
        templatesApi.response.complete(TemplatesResult.SessionExpired)

        assertEquals(StartWorkoutUiState(sessionExpired = true), viewModel.uiState)
    }

    @Test
    fun `reload is ignored while a request is in flight`() {
        viewModel.loadTemplates()

        assertEquals(listOf("session-123"), templatesApi.calls)
    }

    @Test
    fun `retrying after an error clears the error and loads again`() {
        viewModel
        templatesApi.response.complete(TemplatesResult.NetworkError)
        templatesApi.response = CompletableDeferred()

        viewModel.loadTemplates()

        assertEquals(StartWorkoutUiState(isLoading = true), viewModel.uiState)
        templatesApi.response.complete(TemplatesResult.Success(templates))
        assertEquals(StartWorkoutUiState(templates = templates), viewModel.uiState)
        assertEquals(listOf("session-123", "session-123"), templatesApi.calls)
    }

    @Test
    fun `a workout is in progress once it has an exercise`() {
        assertFalse(viewModel.uiState.workoutInProgress)

        activeWorkoutRepository.update { listOf(WorkoutExerciseEntry(squat)) }

        assertTrue(viewModel.uiState.workoutInProgress)
    }

    @Test
    fun `a workout that hasn't been read yet is not in progress`() {
        activeWorkoutRepository.exercises.value = null

        assertFalse(viewModel.uiState.workoutInProgress)
    }

    @Test
    fun `the workout stays in progress when the templates arrive`() {
        activeWorkoutRepository.exercises.value = listOf(WorkoutExerciseEntry(squat))
        viewModel

        templatesApi.response.complete(TemplatesResult.Success(templates))

        assertEquals(StartWorkoutUiState(templates = templates, workoutInProgress = true), viewModel.uiState)
    }

    @Test
    fun `discarding the workout empties it`() {
        activeWorkoutRepository.exercises.value = listOf(WorkoutExerciseEntry(squat))

        viewModel.discardWorkout()

        assertEquals(emptyList<WorkoutExerciseEntry>(), activeWorkoutRepository.exercises.value)
        assertFalse(viewModel.uiState.workoutInProgress)
    }

    private fun assertErrorFor(result: TemplatesResult, expectedMessage: Int) {
        viewModel
        templatesApi.response.complete(result)

        assertEquals(StartWorkoutUiState(errorMessage = expectedMessage), viewModel.uiState)
    }
}
