package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.annotation.StringRes
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasProgressBarRangeInfo
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class WorkoutContentTest {

    @get:Rule
    val composeRule = createComposeRule()

    private val benchPress = Exercise(1, "Bench Press (Barbell)")
    private val squat = Exercise(2, "Squat (Barbell)")

    private var retryClicks = 0
    private val selectedExercises = mutableListOf<Exercise>()

    private fun setContent(uiState: WorkoutUiState) {
        composeRule.setContent {
            GymProgressTrackerTheme {
                WorkoutContent(
                    uiState = uiState,
                    onRetryExercises = { retryClicks++ },
                    onExerciseSelected = { selectedExercises += it }
                )
            }
        }
    }

    private fun str(@StringRes id: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun openExercisePicker() {
        composeRule.onNodeWithText(str(R.string.workout_add_exercise)).performClick()
    }

    @Test
    fun exercisesAreHiddenUntilAddExerciseIsClicked() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))

        composeRule.onNodeWithText("Bench Press (Barbell)").assertDoesNotExist()

        openExercisePicker()

        composeRule.onNodeWithText(str(R.string.workout_choose_exercise)).assertIsDisplayed()
        composeRule.onNodeWithText("Bench Press (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText("Squat (Barbell)").assertIsDisplayed()
    }

    @Test
    fun choosingAnExerciseReportsItAndClosesThePicker() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))
        openExercisePicker()

        composeRule.onNodeWithText("Squat (Barbell)").performClick()

        assertEquals(listOf(squat), selectedExercises)
        composeRule.onNodeWithText(str(R.string.workout_choose_exercise)).assertDoesNotExist()
    }

    @Test
    fun loadingShowsAProgressIndicator() {
        setContent(WorkoutUiState(isLoadingExercises = true))
        openExercisePicker()

        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertIsDisplayed()
    }

    @Test
    fun errorShowsMessageAndRetries() {
        setContent(WorkoutUiState(exercisesErrorMessage = R.string.workout_exercises_error_network))
        openExercisePicker()

        composeRule.onNodeWithText(str(R.string.workout_exercises_error_network)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.workout_exercises_retry)).performClick()

        assertEquals(1, retryClicks)
    }
}
