package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.annotation.StringRes
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.espresso.Espresso
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesApi
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesResult
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesResult
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutViewModel
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.update
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/** Drives the real start and workout screens over one workout repository, switching between them like MainActivity. */
@RunWith(AndroidJUnit4::class)
class WorkoutScreenTest {

    @get:Rule
    val composeRule = createComposeRule()

    private class FakeActiveWorkoutRepository : ActiveWorkoutRepository {
        override val exercises = MutableStateFlow<List<WorkoutExerciseEntry>?>(emptyList())

        override fun update(transform: (List<WorkoutExerciseEntry>) -> List<WorkoutExerciseEntry>) {
            exercises.update { it?.let(transform) }
        }
    }

    private val sessionId = "session-123"
    private val squat = Exercise(1, "Squat (Barbell)")

    private var inWorkout by mutableStateOf(true)
    private lateinit var workoutViewModel: WorkoutViewModel

    @Before
    fun setUp() {
        val activeWorkoutRepository = FakeActiveWorkoutRepository()
        val templatesApi = object : TemplatesApi {
            override suspend fun getTemplates(sessionId: String) = TemplatesResult.Success(emptyList())
        }
        val exercisesApi = object : ExercisesApi {
            override suspend fun getExercises(sessionId: String) = ExercisesResult.Success(emptyList())
        }
        // On the main thread, like viewModel() would, since both update their state as they start.
        val startWorkoutViewModel = composeRule.runOnUiThread {
            StartWorkoutViewModel(templatesApi, activeWorkoutRepository, sessionId)
        }
        workoutViewModel = composeRule.runOnUiThread {
            WorkoutViewModel(exercisesApi, activeWorkoutRepository, sessionId)
        }

        composeRule.setContent {
            GymProgressTrackerTheme {
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
                        onStartNewWorkout = { inWorkout = true },
                        onContinueWorkout = { inWorkout = true },
                        onStartFromTemplate = {},
                        viewModel = startWorkoutViewModel
                    )
                }
            }
        }
    }

    private fun str(@StringRes id: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun addSquat() = composeRule.runOnIdle { workoutViewModel.addExercise(squat) }

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
