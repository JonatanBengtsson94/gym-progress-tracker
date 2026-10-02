package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.annotation.StringRes
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.ViewModelStore
import androidx.lifecycle.ViewModelStoreOwner
import androidx.lifecycle.viewmodel.compose.LocalViewModelStoreOwner
import androidx.test.espresso.Espresso
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/** Drives the real start and workout screens the way MainActivity wires them together. */
@RunWith(AndroidJUnit4::class)
class WorkoutScreenTest {

    @get:Rule
    val composeRule = createComposeRule()

    private val sessionId = "session-123"
    private val squat = Exercise(1, "Squat (Barbell)")

    private val storeOwner = object : ViewModelStoreOwner {
        override val viewModelStore = ViewModelStore()
    }

    private var inWorkout by mutableStateOf(true)

    @Before
    fun setUp() {
        composeRule.setContent {
            CompositionLocalProvider(LocalViewModelStoreOwner provides storeOwner) {
                GymProgressTrackerTheme {
                    val workoutViewModel = workoutViewModel(sessionId)
                    if (inWorkout) {
                        WorkoutScreen(
                            sessionId = sessionId,
                            onSessionExpired = {},
                            onBack = { inWorkout = false },
                            viewModel = workoutViewModel
                        )
                    } else {
                        StartWorkoutScreen(
                            sessionId = sessionId,
                            onSessionExpired = {},
                            workoutInProgress = workoutViewModel.uiState.workoutInProgress,
                            onStartNewWorkout = { inWorkout = true },
                            onContinueWorkout = { inWorkout = true },
                            onDiscardWorkout = workoutViewModel::discardWorkout,
                            onStartFromTemplate = {}
                        )
                    }
                }
            }
        }
    }

    private fun str(@StringRes id: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun addSquat() = composeRule.runOnIdle {
        ViewModelProvider(storeOwner)[workoutViewModelKey(sessionId), WorkoutViewModel::class.java].addExercise(squat)
    }

    @Test
    fun leavingAWorkoutInProgressOffersToContinueIt() {
        addSquat()

        Espresso.pressBack()

        composeRule.onNodeWithText(str(R.string.start_workout_new)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.start_workout_continue)).performClick()
        composeRule.onNodeWithText(squat.name).assertIsDisplayed()
    }

    @Test
    fun leavingAnEmptyWorkoutStartsAfreshNextTime() {
        Espresso.pressBack()

        composeRule.onNodeWithText(str(R.string.start_workout_continue)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).assertIsDisplayed()
    }

    @Test
    fun aDiscardedWorkoutIsGone() {
        addSquat()
        Espresso.pressBack()

        composeRule.onNodeWithText(str(R.string.start_workout_discard)).performClick()
        composeRule.onNodeWithText(str(R.string.start_workout_discard_confirm)).performClick()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).performClick()

        composeRule.onNodeWithText(squat.name).assertDoesNotExist()
    }
}
