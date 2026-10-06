package com.jonatanbengtsson.gymprogresstracker.ui.workout

import androidx.annotation.StringRes
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.remember
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.ui.NavDisplay
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.espresso.Espresso
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkout
import com.jonatanbengtsson.gymprogresstracker.data.ActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.ApiResult
import com.jonatanbengtsson.gymprogresstracker.data.ApiWorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.ExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.FinishedWorkout
import com.jonatanbengtsson.gymprogresstracker.data.RefreshResult
import com.jonatanbengtsson.gymprogresstracker.data.RoomPendingWorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.RoomTemplatesRepository
import com.jonatanbengtsson.gymprogresstracker.data.TemplatesApi
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsApi
import com.jonatanbengtsson.gymprogresstracker.data.local.GymDatabase
import com.jonatanbengtsson.gymprogresstracker.data.testId
import com.jonatanbengtsson.gymprogresstracker.ui.navigation.Screen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutScreen
import com.jonatanbengtsson.gymprogresstracker.ui.start.StartWorkoutViewModel
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.update
import org.junit.After
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import kotlin.uuid.Uuid

/**
 * Drives the real start and workout screens, navigating between them like AppNavigation, over one
 * workout in progress and the real templates and saved workouts repositories on an in-memory database.
 */
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

    private val squat = Exercise(testId(1), "Squat (Barbell)")

    private lateinit var database: GymDatabase
    private lateinit var workoutViewModel: WorkoutViewModel

    @Before
    fun setUp() {
        val activeWorkoutRepository = FakeActiveWorkoutRepository()
        database = Room.inMemoryDatabaseBuilder(ApplicationProvider.getApplicationContext(), GymDatabase::class.java).build()
        val pendingWorkoutsRepository = RoomPendingWorkoutsRepository(database.pendingWorkoutDao())
        // Offline, so whatever is saved stays waiting to sync.
        val templatesApi = object : TemplatesApi {
            override suspend fun getTemplates() = ApiResult.NetworkError
            override suspend fun createTemplate(templateId: Uuid, name: String) = ApiResult.NetworkError
        }
        val workoutsApi = object : WorkoutsApi {
            override suspend fun putWorkout(workoutId: Uuid, templateId: Uuid, workout: FinishedWorkout) = ApiResult.NetworkError
        }
        val templatesRepository = RoomTemplatesRepository(templatesApi, database.templateDao(), pendingWorkoutsRepository)
        val workoutsRepository = ApiWorkoutsRepository(workoutsApi, templatesApi, pendingWorkoutsRepository, templatesRepository)
        val exercisesRepository = object : ExercisesRepository {
            override val exercises = MutableStateFlow(emptyList<Exercise>())
            override suspend fun fetchGlobalIfNoneStored() {}
            override suspend fun refresh() = RefreshResult.Success
        }
        // On the main thread, like viewModel() would, since both update their state as they start.
        val startWorkoutViewModel = composeRule.runOnUiThread {
            StartWorkoutViewModel(templatesRepository, activeWorkoutRepository, workoutsRepository)
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
                            WorkoutScreen(
                                onWorkoutSaved = { backStack.remove(Screen.Workout) },
                                onWorkoutDiscarded = { backStack.remove(Screen.Workout) },
                                viewModel = workoutViewModel
                            )
                        }
                    }
                )
            }
        }
    }

    @After
    fun tearDown() {
        database.close()
    }

    private fun str(@StringRes id: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun waitingToSync(count: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.resources.getQuantityString(R.plurals.start_workout_pending, count, count)

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
        // Room saves off the main thread, which the test doesn't wait for by itself.
        composeRule.waitUntil { composeRule.onAllNodesWithText(waitingToSync(1)).fetchSemanticsNodes().isNotEmpty() }
        composeRule.onNodeWithText(waitingToSync(1)).assertIsDisplayed()
    }

    @Test
    fun aWorkoutSavedOfflineUnderANewNameShowsAsATemplate() {
        composeRule.runOnIdle {
            workoutViewModel.updateName("Leg day")
            workoutViewModel.addExercise(squat)
            workoutViewModel.updateReps(squat.id, 0, "5")
            workoutViewModel.toggleSetCompleted(squat.id, 0)
        }

        composeRule.onNodeWithText(str(R.string.workout_save)).performClick()

        composeRule.waitUntil { composeRule.onAllNodesWithText("Leg day").fetchSemanticsNodes().isNotEmpty() }
        composeRule.onNodeWithText("Leg day").assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.start_workout_no_templates)).assertDoesNotExist()
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
        composeRule.onNodeWithText(waitingToSync(1)).assertDoesNotExist()
    }

    @Test
    fun discardingFromTheWorkoutReturnsToTheStartScreenWithNothingInProgress() {
        addSquat()

        composeRule.onNodeWithText(str(R.string.discard_workout)).performClick()
        composeRule.onNode(hasText(str(R.string.discard_workout_confirm)) and hasAnyAncestor(isDialog())).performClick()

        composeRule.onNodeWithText(str(R.string.start_workout_continue)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).performClick()
        composeRule.onNodeWithText(squat.name).assertDoesNotExist()
    }

    @Test
    fun aDiscardedWorkoutIsGone() {
        addSquat()
        Espresso.pressBack()

        composeRule.onNodeWithText(str(R.string.discard_workout)).performClick()
        composeRule.onNode(hasText(str(R.string.discard_workout_confirm)) and hasAnyAncestor(isDialog())).performClick()
        composeRule.onNodeWithText(str(R.string.start_workout_new)).performClick()

        composeRule.onNodeWithText(squat.name).assertDoesNotExist()
    }
}
