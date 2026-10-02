package com.jonatanbengtsson.gymprogresstracker.ui.workout

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesApi
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesResult
import com.jonatanbengtsson.gymprogresstracker.ui.login.MainDispatcherRule
import kotlinx.coroutines.CompletableDeferred
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

class WorkoutViewModelTest {

    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    /** Suspends each request until the test completes [response]. */
    private class FakeExercisesApi : ExercisesApi {
        var response = CompletableDeferred<ExercisesResult>()
        val calls = mutableListOf<String>()

        override suspend fun getExercises(sessionId: String): ExercisesResult {
            calls += sessionId
            return response.await()
        }
    }

    private val benchPress = Exercise(2, "Bench Press (Barbell)")
    private val squat = Exercise(1, "Squat (Barbell)")

    private val exercisesApi = FakeExercisesApi()

    // Created lazily so each test can set up the fake before the view model loads on init.
    private val viewModel by lazy { WorkoutViewModel(exercisesApi, "session-123") }

    @Test
    fun `loads exercises with the session id on creation`() {
        viewModel

        assertEquals(listOf("session-123"), exercisesApi.calls)
        assertEquals(WorkoutUiState(isLoadingExercises = true), viewModel.uiState)
    }

    @Test
    fun `success exposes the exercises in order`() {
        viewModel
        exercisesApi.response.complete(ExercisesResult.Success(listOf(benchPress, squat)))

        assertEquals(WorkoutUiState(exercises = listOf(benchPress, squat)), viewModel.uiState)
    }

    @Test
    fun `network error shows network error`() {
        assertErrorFor(ExercisesResult.NetworkError, R.string.workout_exercises_error_network)
    }

    @Test
    fun `server error shows server error`() {
        assertErrorFor(ExercisesResult.ServerError, R.string.workout_exercises_error_server)
    }

    @Test
    fun `expired session is reported`() {
        viewModel
        exercisesApi.response.complete(ExercisesResult.SessionExpired)

        assertEquals(WorkoutUiState(sessionExpired = true), viewModel.uiState)
    }

    @Test
    fun `reload is ignored while a request is in flight`() {
        viewModel.loadExercises()

        assertEquals(listOf("session-123"), exercisesApi.calls)
    }

    @Test
    fun `retrying after an error clears the error and loads again`() {
        viewModel
        exercisesApi.response.complete(ExercisesResult.NetworkError)
        exercisesApi.response = CompletableDeferred()

        viewModel.loadExercises()

        assertEquals(WorkoutUiState(isLoadingExercises = true), viewModel.uiState)
        exercisesApi.response.complete(ExercisesResult.Success(listOf(squat)))
        assertEquals(WorkoutUiState(exercises = listOf(squat)), viewModel.uiState)
        assertEquals(listOf("session-123", "session-123"), exercisesApi.calls)
    }

    private fun assertErrorFor(result: ExercisesResult, expectedMessage: Int) {
        viewModel
        exercisesApi.response.complete(result)

        assertEquals(WorkoutUiState(exercisesErrorMessage = expectedMessage), viewModel.uiState)
    }
}
