package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.annotation.StringRes
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.remember
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.espresso.Espresso
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.ui.NavDisplay
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkout
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.ApiResult
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.FinishedWorkout
import com.jonatanbengtsson.gymprogresstracker.data.RefreshResult
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutTemplate
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.testId
import com.jonatanbengtsson.gymprogresstracker.ui.navigation.Screen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutViewModel
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.update
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import kotlin.uuid.Uuid

/** Drives the real start and workout screens over one workout repository, navigating between them like AppNavigation. */
@RunWith(AndroidJUnit4::class)
class WorkoutScreenTest {

    @get:Rule
    val composeRule = createComposeRule()

    private class FakeActiveWorkoutRepository : ActiveWorkoutRepository {
        override val workout = MutableStateFlow<ActiveWorkout?>(ActiveWorkout())

        override fun update(transform: (ActiveWorkout) -> ActiveWorkout) {
            workout.update { it?.let(transform) }
        }
    }

    private class FakeSessionRepository : SessionRepository {
        override val session = MutableStateFlow<SessionState>(SessionState.LoggedIn("session-123"))

        override suspend fun logIn(username: String, sessionId: String) {
            session.value = SessionState.LoggedIn(sessionId)
        }

        override suspend fun endSession(sessionId: String) {
            session.value = SessionState.LoggedOut
        }

        override suspend fun logOut() {
            session.value = SessionState.LoggedOut
        }
    }

    private val squat = Exercise(testId(1), "Squat (Barbell)")

    private lateinit var workoutViewModel: WorkoutViewModel

    @Before
    fun setUp() {
        val activeWorkoutRepository = FakeActiveWorkoutRepository()
        val sessionRepository = FakeSessionRepository()
        val templatesRepository = object : TemplatesRepository {
            override val templates = MutableStateFlow(emptyList<WorkoutTemplate>())
            override suspend fun refresh() = RefreshResult.Success
        }
        val workoutsRepository = object : WorkoutsRepository {
            override suspend fun save(workoutId: Uuid, workout: FinishedWorkout) = ApiResult.Success(Unit)
        }
        val exercisesRepository = object : ExercisesRepository {
            override val exercises = MutableStateFlow(emptyList<Exercise>())
            override suspend fun refresh() = RefreshResult.Success
        }
        // On the main thread, like viewModel() would, since both update their state as they start.
        val startWorkoutViewModel = composeRule.runOnUiThread {
            StartWorkoutViewModel(templatesRepository, activeWorkoutRepository, sessionRepository)
        }
        workoutViewModel = composeRule.runOnUiThread {
            WorkoutViewModel(exercisesRepository, activeWorkoutRepository, workoutsRepository)
        }

        composeRule.setContent {
            GymProgressTrackerTheme {
                val backStack = remember { mutableStateListOf<NavKey>(Screen.StartWorkout, Screen.Workout) }
                NavDisplay(
                    backStack = backStack,
                    entryProvider = entryProvider {
                        entry<Screen.StartWorkout> {
                            StartWorkoutScreen(
                                onStartNewWorkout = { backStack.add(Screen.Workout) },
                                onContinueWorkout = { backStack.add(Screen.Workout) },
                                onStartFromTemplate = {},
                                viewModel = startWorkoutViewModel
                            )
                        }
                        entry<Screen.Workout> {
                            WorkoutScreen(onWorkoutSaved = { backStack.remove(Screen.Workout) }, viewModel = workoutViewModel)
                        }
                    }
                )
            }
        }
    }

    private fun str(@StringRes id: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun addSquat() = composeRule.runOnIdle { workoutViewModel.addExercise(squat) }

    @Test
    fun aSavedWorkoutReturnsToTheStartScreenWithNothingInProgress() {
        composeRule.runOnIdle {
            workoutViewModel.updateName("Leg day")
            workoutViewModel.addExercise(squat)
            workoutViewModel.updateReps(squat.id, 0, "5")
            workoutViewModel.toggleSetCompleted(squat.id, 0)
        }

        composeRule.onNodeWithText(str(R.string.workout_save)).performClick()

        composeRule.onNodeWithText(str(R.string.start_workout_continue)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).assertIsDisplayed()
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
    fun startingANewWorkoutShowsHowLongItHasRun() {
        Espresso.pressBack()

        composeRule.onNodeWithText(str(R.string.start_workout_new)).performClick()

        composeRule.onNodeWithText("0:0", substring = true).assertIsDisplayed()
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
