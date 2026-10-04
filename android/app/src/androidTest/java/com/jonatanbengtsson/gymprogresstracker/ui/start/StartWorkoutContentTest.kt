package com.jonatanbengtsson.gymprogresstracker.ui.start

import androidx.annotation.StringRes
import androidx.compose.runtime.Composable
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasProgressBarRangeInfo
import androidx.compose.ui.test.junit4.StateRestorationTester
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.LatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExercise
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutSet
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.data.testId
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

@RunWith(AndroidJUnit4::class)
class StartWorkoutContentTest {

    @get:Rule
    val composeRule = createComposeRule()

    private val pushDay = WorkoutTemplate(
        id = testId(1),
        name = "Push Day",
        latestWorkout = LatestWorkout(
            workoutId = testId(7),
            startedAt = Instant.parse("2024-05-08T09:00:00Z"),
            completedAt = Instant.parse("2024-05-08T10:00:00Z"),
            exercises = listOf(
                WorkoutExercise(testId(1), "Bench Press (Barbell)", listOf(WorkoutSet(8, 60000))),
                WorkoutExercise(testId(2), "Overhead Press (Barbell)", listOf(WorkoutSet(10, 30000)))
            )
        )
    )
    private val legDay = WorkoutTemplate(id = testId(2), name = "Leg Day", latestWorkout = null)
    private val pullDay = WorkoutTemplate(
        id = testId(3),
        name = "Pull Day",
        latestWorkout = LatestWorkout(
            workoutId = testId(8),
            startedAt = Instant.parse("2024-05-09T09:00:00Z"),
            completedAt = Instant.parse("2024-05-09T10:00:00Z"),
            exercises = listOf(WorkoutExercise(testId(3), "Deadlift (Barbell)", listOf(WorkoutSet(5, 100000))))
        )
    )
    private val emptyDay = WorkoutTemplate(
        id = testId(4),
        name = "Empty Day",
        latestWorkout = LatestWorkout(
            workoutId = testId(9),
            startedAt = Instant.parse("2024-05-10T09:00:00Z"),
            completedAt = Instant.parse("2024-05-10T10:00:00Z"),
            exercises = emptyList()
        )
    )

    private var newWorkoutClicks = 0
    private var continueClicks = 0
    private var discardClicks = 0
    private var retryClicks = 0
    private var logOutClicks = 0
    private val startedTemplates = mutableListOf<WorkoutTemplate>()

    private fun setContent(uiState: StartWorkoutUiState) {
        composeRule.setContent { Content(uiState) }
    }

    @Composable
    private fun Content(uiState: StartWorkoutUiState) {
        GymProgressTrackerTheme {
            StartWorkoutContent(
                uiState = uiState,
                onRetry = { retryClicks++ },
                onStartNewWorkout = { newWorkoutClicks++ },
                onContinueWorkout = { continueClicks++ },
                onDiscardWorkout = { discardClicks++ },
                onStartFromTemplate = { startedTemplates += it },
                onLogOut = { logOutClicks++ }
            )
        }
    }

    @Test
    fun withoutAWorkoutInProgressThereIsNothingToContinue() {
        setContent(StartWorkoutUiState())

        composeRule.onNodeWithText(str(R.string.start_workout_new)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_in_progress)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.start_workout_continue)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.start_workout_discard)).assertDoesNotExist()
    }

    @Test
    fun aWorkoutInProgressCanBeContinuedInsteadOfStartingANewOne() {
        setContent(StartWorkoutUiState(workoutInProgress = true))

        composeRule.onNodeWithText(str(R.string.start_workout_in_progress)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.start_workout_continue)).performClick()

        assertEquals(1, continueClicks)
        assertEquals(0, newWorkoutClicks)
    }

    @Test
    fun discardingAWorkoutAsksFirst() {
        setContent(StartWorkoutUiState(workoutInProgress = true))

        composeRule.onNodeWithText(str(R.string.start_workout_discard)).performClick()
        composeRule.onNodeWithText(str(R.string.start_workout_discard_title)).assertIsDisplayed()
        assertEquals(0, discardClicks)

        composeRule.onNodeWithText(str(R.string.start_workout_discard_confirm)).performClick()

        assertEquals(1, discardClicks)
        composeRule.onNodeWithText(str(R.string.start_workout_discard_title)).assertDoesNotExist()
    }

    @Test
    fun cancellingTheDiscardKeepsTheWorkout() {
        setContent(StartWorkoutUiState(workoutInProgress = true))

        composeRule.onNodeWithText(str(R.string.start_workout_discard)).performClick()
        composeRule.onNodeWithText(str(R.string.start_workout_discard_cancel)).performClick()

        assertEquals(0, discardClicks)
        composeRule.onNodeWithText(str(R.string.start_workout_discard_title)).assertDoesNotExist()
    }

    private fun str(@StringRes id: Int, vararg formatArgs: Any) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id, *formatArgs)

    @Test
    fun clickingLogOutLogsOut() {
        setContent(StartWorkoutUiState())

        composeRule.onNodeWithText(str(R.string.start_workout_log_out)).performClick()

        assertEquals(1, logOutClicks)
    }

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
        composeRule.onNodeWithText("2 exercises").assertIsDisplayed()
        composeRule.onNodeWithText("Leg Day").assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_never_performed)).assertIsDisplayed()
    }

    @Test
    fun expandingATemplateListsItsExercisesWithoutStartingIt() {
        setContent(StartWorkoutUiState(templates = listOf(pushDay, legDay)))

        composeRule.onNodeWithText("Bench Press (Barbell)").assertDoesNotExist()

        composeRule.onNodeWithText("2 exercises").performClick()

        composeRule.onNodeWithText("Bench Press (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText("Overhead Press (Barbell)").assertIsDisplayed()
        assertEquals(emptyList<WorkoutTemplate>(), startedTemplates)

        composeRule.onNodeWithText("2 exercises").performClick()

        composeRule.onNodeWithText("Bench Press (Barbell)").assertDoesNotExist()
    }

    @Test
    fun templatesWithoutExercisesHaveNothingToExpand() {
        setContent(StartWorkoutUiState(templates = listOf(legDay, emptyDay)))

        composeRule.onNodeWithText("Leg Day").assertIsDisplayed()
        composeRule.onNodeWithText("Empty Day").assertIsDisplayed()
        composeRule.onAllNodesWithText("exercise", substring = true).assertCountEquals(0)
    }

    @Test
    fun expandingATemplateLeavesTheOthersCollapsed() {
        setContent(StartWorkoutUiState(templates = listOf(pushDay, pullDay)))

        composeRule.onNodeWithText("2 exercises").performClick()

        composeRule.onNodeWithText("Bench Press (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText("Deadlift (Barbell)").assertDoesNotExist()
    }

    @Test
    fun expandedTemplateStaysExpandedAfterStateRestore() {
        val restorationTester = StateRestorationTester(composeRule)
        restorationTester.setContent { Content(StartWorkoutUiState(templates = listOf(pushDay))) }

        composeRule.onNodeWithText("2 exercises").performClick()
        restorationTester.emulateSavedInstanceStateRestore()

        composeRule.onNodeWithText("Bench Press (Barbell)").assertIsDisplayed()
    }

    @Test
    fun performedTemplateShowsWhenItWasLastPerformed() {
        setContent(StartWorkoutUiState(templates = listOf(pushDay)))

        // Formatted the same way as the screen so the test passes in any locale and time zone.
        val date = DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM)
            .withZone(ZoneId.systemDefault())
            .format(pushDay.latestWorkout!!.completedAt)
        composeRule.onNodeWithText(str(R.string.start_workout_last_performed, date)).assertIsDisplayed()
    }

    @Test
    fun newWorkoutIsAvailableWhileTemplatesLoad() {
        setContent(StartWorkoutUiState(isLoading = true))

        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).performClick()

        assertEquals(1, newWorkoutClicks)
    }

    @Test
    fun storedTemplatesShowWhileTheyAreFetched() {
        setContent(StartWorkoutUiState(isLoading = true, templates = listOf(pushDay)))

        composeRule.onNodeWithText(pushDay.name).assertIsDisplayed()
        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertDoesNotExist()
    }

    @Test
    fun storedTemplatesShowWhenFetchingThemFails() {
        setContent(StartWorkoutUiState(templates = listOf(pushDay), errorMessage = R.string.start_workout_error_network))

        composeRule.onNodeWithText(pushDay.name).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_error_network)).assertDoesNotExist()
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
