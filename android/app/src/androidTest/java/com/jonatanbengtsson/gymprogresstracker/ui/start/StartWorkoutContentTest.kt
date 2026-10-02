package com.jonatanbengtsson.gymprogresstracker.ui.start

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
import com.jonatanbengtsson.gymprogresstracker.data.LatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExercise
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutSet
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.time.Instant

@RunWith(AndroidJUnit4::class)
class StartWorkoutContentTest {

    @get:Rule
    val composeRule = createComposeRule()

    private val pushDay = WorkoutTemplate(
        id = 1,
        name = "Push Day",
        latestWorkout = LatestWorkout(
            workoutId = 7,
            startedAt = Instant.parse("2024-05-08T09:00:00Z"),
            completedAt = Instant.parse("2024-05-08T10:00:00Z"),
            exercises = listOf(
                WorkoutExercise(1, "Bench Press (Barbell)", listOf(WorkoutSet(8, 60000))),
                WorkoutExercise(2, "Overhead Press (Barbell)", listOf(WorkoutSet(10, 30000)))
            )
        )
    )
    private val legDay = WorkoutTemplate(id = 2, name = "Leg Day", latestWorkout = null)

    private var newWorkoutClicks = 0
    private var retryClicks = 0
    private val startedTemplates = mutableListOf<WorkoutTemplate>()

    private fun setContent(uiState: StartWorkoutUiState) {
        composeRule.setContent {
            GymProgressTrackerTheme {
                StartWorkoutContent(
                    uiState = uiState,
                    onRetry = { retryClicks++ },
                    onStartNewWorkout = { newWorkoutClicks++ },
                    onStartFromTemplate = { startedTemplates += it }
                )
            }
        }
    }

    private fun str(@StringRes id: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    @Test
    fun clickingNewWorkoutStartsAnEmptyWorkout() {
        setContent(StartWorkoutUiState(templates = listOf(pushDay)))

        composeRule.onNodeWithText(str(R.string.start_workout_new)).performClick()

        assertEquals(1, newWorkoutClicks)
        assertEquals(emptyList<WorkoutTemplate>(), startedTemplates)
    }

    @Test
    fun clickingATemplateStartsFromThatTemplate() {
        setContent(StartWorkoutUiState(templates = listOf(pushDay, legDay)))

        composeRule.onNodeWithText("Leg Day").performClick()

        assertEquals(listOf(legDay), startedTemplates)
        assertEquals(0, newWorkoutClicks)
    }

    @Test
    fun templatesShowTheirLatestWorkout() {
        setContent(StartWorkoutUiState(templates = listOf(pushDay, legDay)))

        composeRule.onNodeWithText("Push Day").assertIsDisplayed()
        composeRule.onNodeWithText("Bench Press (Barbell) · Overhead Press (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText("Leg Day").assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_never_performed)).assertIsDisplayed()
    }

    @Test
    fun newWorkoutIsAvailableWhileTemplatesLoad() {
        setContent(StartWorkoutUiState(isLoading = true))

        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).performClick()

        assertEquals(1, newWorkoutClicks)
    }

    @Test
    fun errorShowsMessageAndRetry() {
        setContent(StartWorkoutUiState(errorMessage = R.string.start_workout_error_network))

        composeRule.onNodeWithText(str(R.string.start_workout_error_network)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_retry)).performClick()

        assertEquals(1, retryClicks)
    }

    @Test
    fun noTemplatesShowsEmptyMessage() {
        setContent(StartWorkoutUiState())

        composeRule.onNodeWithText(str(R.string.start_workout_no_templates)).assertIsDisplayed()
    }
}
