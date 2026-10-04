package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.annotation.StringRes
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.test.assert
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.assertIsOff
import androidx.compose.ui.test.assertIsOn
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasProgressBarRangeInfo
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.espresso.Espresso
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.SetEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.data.testId
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.time.Instant

@RunWith(AndroidJUnit4::class)
class WorkoutContentTest {

    @get:Rule
    val composeRule = createComposeRule()

    private val benchPress = Exercise(testId(1), "Bench Press (Barbell)")
    private val squat = Exercise(testId(2), "Squat (Barbell)")

    private var retryClicks = 0
    private val selectedExercises = mutableListOf<Exercise>()
    private val setEvents = mutableListOf<String>()

    private fun setContent(uiState: WorkoutUiState) {
        composeRule.setContent {
            GymProgressTrackerTheme {
                WorkoutContent(
                    uiState = uiState,
                    onNameChange = { setEvents += "name $it" },
                    onRetryExercises = { retryClicks++ },
                    onExerciseSelected = { selectedExercises += it },
                    onRemoveExercise = { setEvents += "remove exercise $it" },
                    onAddSet = { setEvents += "add $it" },
                    onRemoveSet = { id, index -> setEvents += "remove $id $index" },
                    onToggleSetCompleted = { id, index -> setEvents += "complete $id $index" },
                    onWeightChange = { id, index, kg -> setEvents += "weight $id $index $kg" },
                    onRepsChange = { id, index, reps -> setEvents += "reps $id $index $reps" }
                )
            }
        }
    }

    private fun str(@StringRes id: Int, vararg formatArgs: Any) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id, *formatArgs)

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
    fun anUnnamedWorkoutShowsAPlaceholder() {
        setContent(WorkoutUiState())

        composeRule.onNodeWithText(str(R.string.workout_name_placeholder)).assertIsDisplayed()
    }

    @Test
    fun typingANameReportsIt() {
        setContent(WorkoutUiState(name = "Push"))

        composeRule.onNodeWithContentDescription(str(R.string.workout_name_description)).assert(hasText("Push"))
        composeRule.onNodeWithContentDescription(str(R.string.workout_name_description)).performTextInput(" day")

        assertEquals(listOf("name Push day"), setEvents)
        composeRule.onNodeWithText(str(R.string.workout_name_placeholder)).assertDoesNotExist()
    }

    @Test
    fun aStartedWorkoutShowsHowLongItHasRun() {
        setContent(WorkoutUiState(startedAt = Instant.now().minusSeconds(3600 + 5 * 60)))

        composeRule.onNodeWithText("1:05:", substring = true).assertIsDisplayed()
    }

    @Test
    fun aWorkoutStillLoadingShowsProgressInsteadOfAddExercise() {
        setContent(WorkoutUiState(isLoadingWorkout = true))

        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.workout_add_exercise)).assertDoesNotExist()
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
    fun backArrowClosesThePickerWithoutChoosing() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))
        openExercisePicker()

        composeRule.onNodeWithContentDescription(str(R.string.workout_picker_back)).performClick()

        composeRule.onNodeWithText(str(R.string.workout_choose_exercise)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.workout_add_exercise)).assertIsDisplayed()
        assertEquals(emptyList<Exercise>(), selectedExercises)
    }

    @Test
    fun systemBackClosesThePickerWithoutChoosing() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))
        openExercisePicker()

        Espresso.pressBack()

        composeRule.onNodeWithText(str(R.string.workout_choose_exercise)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.workout_add_exercise)).assertIsDisplayed()
        assertEquals(emptyList<Exercise>(), selectedExercises)
    }

    @Test
    fun addedExercisesAreShownInTheWorkout() {
        setContent(
            WorkoutUiState(
                workoutExercises = listOf(WorkoutExerciseEntry(squat), WorkoutExerciseEntry(benchPress)),
                exercises = listOf(benchPress, squat)
            )
        )

        composeRule.onNodeWithText("Squat (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText("Bench Press (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.workout_add_exercise)).assertIsDisplayed()
    }

    @Test
    fun exercisesAlreadyInTheWorkoutCannotBePickedAgain() {
        setContent(WorkoutUiState(workoutExercises = listOf(WorkoutExerciseEntry(squat)), exercises = listOf(benchPress, squat)))
        openExercisePicker()

        val pickerSquat = composeRule.onNode(hasText("Squat (Barbell)") and hasClickAction())
        pickerSquat.assertIsNotEnabled()
        pickerSquat.assert(hasContentDescription(str(R.string.workout_exercise_added)))
        composeRule.onNode(hasText("Bench Press (Barbell)") and hasClickAction()).assertIsEnabled()

        pickerSquat.performClick()

        assertEquals(emptyList<Exercise>(), selectedExercises)
        composeRule.onNodeWithText(str(R.string.workout_choose_exercise)).assertIsDisplayed()
    }

    @Test
    fun searchingNarrowsTheListOfExercises() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))
        openExercisePicker()

        composeRule.onNode(hasSetTextAction()).performTextInput("squ")

        composeRule.onNodeWithText("Squat (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText("Bench Press (Barbell)").assertDoesNotExist()
    }

    @Test
    fun searchWithoutMatchesSaysSo() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))
        openExercisePicker()

        composeRule.onNode(hasSetTextAction()).performTextInput("curl")

        composeRule.onNodeWithText(str(R.string.workout_search_no_results, "curl")).assertIsDisplayed()
    }

    @Test
    fun clearingTheSearchShowsAllExercisesAgain() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))
        openExercisePicker()
        composeRule.onNode(hasSetTextAction()).performTextInput("squ")

        composeRule.onNodeWithContentDescription(str(R.string.workout_search_clear)).performClick()

        composeRule.onNodeWithText("Bench Press (Barbell)").assertIsDisplayed()
        composeRule.onNodeWithText("Squat (Barbell)").assertIsDisplayed()
    }

    @Test
    fun reopeningThePickerStartsWithAnEmptySearch() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat)))
        openExercisePicker()
        composeRule.onNode(hasSetTextAction()).performTextInput("squ")
        composeRule.onNodeWithContentDescription(str(R.string.workout_picker_back)).performClick()

        openExercisePicker()

        composeRule.onNodeWithText("Bench Press (Barbell)").assertIsDisplayed()
    }

    @Test
    fun exerciseCardsShowTheirSets() {
        setContent(
            WorkoutUiState(
                workoutExercises = listOf(
                    WorkoutExerciseEntry(squat, listOf(SetEntry("100", "5"), SetEntry("102,5", "3", id = 1)))
                )
            )
        )

        composeRule.onNodeWithText("100").assertIsDisplayed()
        composeRule.onNodeWithText("102,5").assertIsDisplayed()
        composeRule.onNodeWithText("3").assertIsDisplayed()
        composeRule.onNodeWithContentDescription(str(R.string.workout_remove_set, 2)).assertIsDisplayed()
    }

    @Test
    fun typingInASetReportsTheExerciseSetAndValue() {
        setContent(WorkoutUiState(workoutExercises = listOf(WorkoutExerciseEntry(squat))))

        composeRule.onNodeWithContentDescription(str(R.string.workout_set_weight_description, 1)).performTextInput("100")
        composeRule.onNodeWithContentDescription(str(R.string.workout_set_reps_description, 1)).performTextInput("5")

        assertEquals(listOf("weight ${squat.id} 0 100", "reps ${squat.id} 0 5"), setEvents)
    }

    @Test
    fun addAndRemoveSetReportTheExerciseAndSet() {
        setContent(
            WorkoutUiState(
                workoutExercises = listOf(WorkoutExerciseEntry(squat, listOf(SetEntry(), SetEntry(id = 1))))
            )
        )

        composeRule.onNodeWithText(str(R.string.workout_add_set)).performClick()
        composeRule.onNodeWithContentDescription(str(R.string.workout_remove_set, 2)).performClick()

        assertEquals(listOf("add ${squat.id}", "remove ${squat.id} 1"), setEvents)
    }

    @Test
    fun tickingASetReportsTheExerciseAndSet() {
        setContent(WorkoutUiState(workoutExercises = listOf(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5"))))))

        composeRule.onNodeWithContentDescription(str(R.string.workout_complete_set, 1))
            .assertIsOff()
            .performClick()

        assertEquals(listOf("complete ${squat.id} 0"), setEvents)
    }

    @Test
    fun completedSetsShowAsTicked() {
        setContent(
            WorkoutUiState(
                workoutExercises = listOf(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5", completed = true))))
            )
        )

        composeRule.onNodeWithContentDescription(str(R.string.workout_complete_set, 1)).assertIsOn()
    }

    @Test
    fun setsWithoutRepsCannotBeTicked() {
        setContent(WorkoutUiState(workoutExercises = listOf(WorkoutExerciseEntry(squat))))

        composeRule.onNodeWithContentDescription(str(R.string.workout_complete_set, 1)).assertIsNotEnabled()
    }

    @Test
    fun removingAnExerciseWithoutEnteredSetsDoesNotAsk() {
        setContent(WorkoutUiState(workoutExercises = listOf(WorkoutExerciseEntry(squat))))

        composeRule.onNodeWithContentDescription(str(R.string.workout_remove_exercise, squat.name)).performClick()

        assertEquals(listOf("remove exercise ${squat.id}"), setEvents)
        composeRule.onNodeWithText(str(R.string.workout_remove_exercise_title, squat.name)).assertDoesNotExist()
    }

    @Test
    fun removingAnExerciseWithEnteredSetsAsksFirst() {
        setContent(WorkoutUiState(workoutExercises = listOf(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5"))))))

        composeRule.onNodeWithContentDescription(str(R.string.workout_remove_exercise, squat.name)).performClick()

        composeRule.onNodeWithText(str(R.string.workout_remove_exercise_title, squat.name)).assertIsDisplayed()
        assertEquals(emptyList<String>(), setEvents)

        composeRule.onNodeWithText(str(R.string.workout_remove_exercise_confirm)).performClick()

        assertEquals(listOf("remove exercise ${squat.id}"), setEvents)
        composeRule.onNodeWithText(str(R.string.workout_remove_exercise_title, squat.name)).assertDoesNotExist()
    }

    @Test
    fun cancellingTheRemovalKeepsTheExercise() {
        setContent(WorkoutUiState(workoutExercises = listOf(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5"))))))

        composeRule.onNodeWithContentDescription(str(R.string.workout_remove_exercise, squat.name)).performClick()
        composeRule.onNodeWithText(str(R.string.workout_remove_exercise_cancel)).performClick()

        assertEquals(emptyList<String>(), setEvents)
        composeRule.onNodeWithText(str(R.string.workout_remove_exercise_title, squat.name)).assertDoesNotExist()
    }

    @Test
    fun loadingShowsAProgressIndicator() {
        setContent(WorkoutUiState(isLoadingExercises = true))
        openExercisePicker()

        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertIsDisplayed()
    }

    @Test
    fun storedExercisesShowWhileTheyAreFetched() {
        setContent(WorkoutUiState(isLoadingExercises = true, exercises = listOf(benchPress, squat)))
        openExercisePicker()

        composeRule.onNodeWithText(benchPress.name).assertIsDisplayed()
        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertDoesNotExist()
    }

    @Test
    fun storedExercisesShowWhenFetchingThemFails() {
        setContent(WorkoutUiState(exercises = listOf(benchPress, squat), exercisesErrorMessage = R.string.workout_exercises_error_network))
        openExercisePicker()

        composeRule.onNodeWithText(benchPress.name).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.workout_exercises_error_network)).assertDoesNotExist()
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
