package com.jonatanbengtsson.gymprogresstracker.ui.workout

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.Exercise
import com.jonatanbengtsson.gymprogresstracker.data.FakeActiveWorkoutRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeExercisesRepository
import com.jonatanbengtsson.gymprogresstracker.data.RefreshResult
import com.jonatanbengtsson.gymprogresstracker.data.SetEntry
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutExerciseEntry
import com.jonatanbengtsson.gymprogresstracker.data.testId
import com.jonatanbengtsson.gymprogresstracker.ui.login.MainDispatcherRule
import kotlinx.coroutines.CompletableDeferred
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class WorkoutViewModelTest {

    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val benchPress = Exercise(testId(2), "Bench Press (Barbell)")
    private val deadlift = Exercise(testId(3), "Deadlift")
    private val squat = Exercise(testId(1), "Squat (Barbell)")

    private val exercisesRepository = FakeExercisesRepository()
    private val activeWorkoutRepository = FakeActiveWorkoutRepository()

    // Created lazily so each test can set up the fakes before the view model refreshes on init.
    private val viewModel by lazy { WorkoutViewModel(exercisesRepository, activeWorkoutRepository) }

    @Test
    fun `refreshes the exercises on creation`() {
        viewModel

        assertEquals(1, exercisesRepository.refreshes)
        assertEquals(WorkoutUiState(isLoadingExercises = true), viewModel.uiState)
    }

    @Test
    fun `stored exercises show while they're being refreshed`() {
        exercisesRepository.exercises.value = listOf(benchPress, squat)

        assertEquals(WorkoutUiState(isLoadingExercises = true, exercises = listOf(benchPress, squat)), viewModel.uiState)
    }

    @Test
    fun `refreshed exercises replace the stored ones`() {
        exercisesRepository.exercises.value = listOf(squat)
        viewModel

        exercisesRepository.exercises.value = listOf(benchPress, squat)
        exercisesRepository.refreshResult.complete(RefreshResult.Success)

        assertEquals(WorkoutUiState(exercises = listOf(benchPress, squat)), viewModel.uiState)
    }

    @Test
    fun `network error shows network error`() {
        assertErrorFor(RefreshResult.NetworkError, R.string.workout_exercises_error_network)
    }

    @Test
    fun `server error shows server error`() {
        assertErrorFor(RefreshResult.ServerError, R.string.workout_exercises_error_server)
    }

    @Test
    fun `a failed refresh keeps showing the stored exercises`() {
        exercisesRepository.exercises.value = listOf(benchPress, squat)
        viewModel

        exercisesRepository.refreshResult.complete(RefreshResult.NetworkError)

        assertEquals(
            WorkoutUiState(exercises = listOf(benchPress, squat), exercisesErrorMessage = R.string.workout_exercises_error_network),
            viewModel.uiState
        )
    }

    @Test
    fun `an expired session shows no error`() {
        viewModel
        exercisesRepository.refreshResult.complete(RefreshResult.SessionExpired)

        assertEquals(WorkoutUiState(), viewModel.uiState)
    }

    @Test
    fun `reload is ignored while a refresh is in flight`() {
        viewModel.loadExercises()

        assertEquals(1, exercisesRepository.refreshes)
    }

    @Test
    fun `retrying after an error clears the error and refreshes again`() {
        viewModel
        exercisesRepository.refreshResult.complete(RefreshResult.NetworkError)
        exercisesRepository.refreshResult = CompletableDeferred()

        viewModel.loadExercises()

        assertEquals(WorkoutUiState(isLoadingExercises = true), viewModel.uiState)
        exercisesRepository.exercises.value = listOf(squat)
        exercisesRepository.refreshResult.complete(RefreshResult.Success)
        assertEquals(WorkoutUiState(exercises = listOf(squat)), viewModel.uiState)
        assertEquals(2, exercisesRepository.refreshes)
    }

    @Test
    fun `added exercises are kept in the order they were added`() {
        viewModel.addExercise(squat)
        viewModel.addExercise(benchPress)

        assertEquals(listOf(squat, benchPress), workoutExercises())
    }

    @Test
    fun `adding an exercise that is already in the workout is ignored`() {
        viewModel.addExercise(squat)
        viewModel.addExercise(benchPress)
        viewModel.addExercise(squat)

        assertEquals(listOf(squat, benchPress), workoutExercises())
    }

    @Test
    fun `added exercises survive reloading the exercise list`() {
        viewModel.addExercise(squat)
        exercisesRepository.refreshResult.complete(RefreshResult.NetworkError)
        exercisesRepository.refreshResult = CompletableDeferred()

        viewModel.loadExercises()
        exercisesRepository.exercises.value = listOf(benchPress, squat)
        exercisesRepository.refreshResult.complete(RefreshResult.Success)

        assertEquals(listOf(squat), workoutExercises())
    }

    @Test
    fun `removing an exercise keeps the others in order`() {
        viewModel.addExercise(squat)
        viewModel.addExercise(benchPress)
        viewModel.addExercise(deadlift)

        viewModel.removeExercise(benchPress.id)

        assertEquals(listOf(squat, deadlift), workoutExercises())
    }

    @Test
    fun `a removed exercise can be added again, starting over`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")
        viewModel.removeExercise(squat.id)

        viewModel.addExercise(squat)

        assertEquals(listOf(SetEntry()), setsOf(squat))
    }

    @Test
    fun `an exercise has entered sets once any set has a value or is completed`() {
        assertFalse(WorkoutExerciseEntry(squat).hasEnteredSets)
        assertFalse(WorkoutExerciseEntry(squat, emptyList()).hasEnteredSets)
        assertTrue(WorkoutExerciseEntry(squat, listOf(SetEntry(), SetEntry(weightKg = "60"))).hasEnteredSets)
        assertTrue(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5"))).hasEnteredSets)
    }

    @Test
    fun `the workout is loading until the saved one has been read`() {
        activeWorkoutRepository.exercises.value = null

        assertTrue(viewModel.uiState.isLoadingWorkout)

        activeWorkoutRepository.exercises.value = listOf(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5"))))

        assertFalse(viewModel.uiState.isLoadingWorkout)
        assertEquals(listOf(SetEntry(reps = "5")), setsOf(squat))
    }

    @Test
    fun `changes to the workout are saved`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")

        assertEquals(listOf(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5")))), activeWorkoutRepository.exercises.value)
    }

    @Test
    fun `a workout discarded elsewhere is gone, but the exercises to pick from are kept`() {
        exercisesRepository.exercises.value = listOf(benchPress, squat)
        exercisesRepository.refreshResult.complete(RefreshResult.Success)
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")

        activeWorkoutRepository.update { emptyList() }

        assertEquals(WorkoutUiState(exercises = listOf(benchPress, squat)), viewModel.uiState)
    }

    @Test
    fun `an added exercise starts with one empty set`() {
        viewModel.addExercise(squat)

        assertEquals(listOf(SetEntry()), setsOf(squat))
    }

    @Test
    fun `adding a set copies the previous one`() {
        viewModel.addExercise(squat)
        viewModel.updateWeight(squat.id, 0, "100")
        viewModel.updateReps(squat.id, 0, "5")

        viewModel.addSet(squat.id)

        assertEquals(listOf(SetEntry("100", "5"), SetEntry("100", "5", id = 1)), setsOf(squat))
    }

    @Test
    fun `adding a set after removing them all starts empty`() {
        viewModel.addExercise(squat)
        viewModel.updateWeight(squat.id, 0, "100")
        viewModel.removeSet(squat.id, 0)

        viewModel.addSet(squat.id)

        assertEquals(listOf(SetEntry()), setsOf(squat))
    }

    @Test
    fun `removing a set keeps the others in order`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")
        viewModel.addSet(squat.id)
        viewModel.updateReps(squat.id, 1, "4")
        viewModel.addSet(squat.id)
        viewModel.updateReps(squat.id, 2, "3")

        viewModel.removeSet(squat.id, 1)

        assertEquals(listOf(SetEntry(reps = "5"), SetEntry(reps = "3", id = 2)), setsOf(squat))
    }

    @Test
    fun `a set added after removing one gets an id no other set has`() {
        viewModel.addExercise(squat)
        viewModel.addSet(squat.id)
        viewModel.addSet(squat.id)
        viewModel.removeSet(squat.id, 0)

        viewModel.addSet(squat.id)

        assertEquals(listOf(1, 2, 3), setsOf(squat).map { it.id })
    }

    @Test
    fun `a set with reps can be completed and un-completed`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")

        viewModel.toggleSetCompleted(squat.id, 0)
        assertEquals(SetEntry(reps = "5", completed = true), setsOf(squat).single())

        viewModel.toggleSetCompleted(squat.id, 0)
        assertEquals(SetEntry(reps = "5"), setsOf(squat).single())
    }

    @Test
    fun `a set without reps cannot be completed`() {
        viewModel.addExercise(squat)

        viewModel.toggleSetCompleted(squat.id, 0)
        assertEquals(SetEntry(), setsOf(squat).single())

        viewModel.updateReps(squat.id, 0, "0")
        viewModel.toggleSetCompleted(squat.id, 0)
        assertEquals(SetEntry(reps = "0"), setsOf(squat).single())
    }

    @Test
    fun `changing the reps of a completed set keeps it completed`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")
        viewModel.toggleSetCompleted(squat.id, 0)

        viewModel.updateReps(squat.id, 0, "6")
        viewModel.updateWeight(squat.id, 0, "100")

        assertEquals(SetEntry("100", "6", completed = true), setsOf(squat).single())
    }

    @Test
    fun `clearing the reps of a completed set un-completes it`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")
        viewModel.toggleSetCompleted(squat.id, 0)

        viewModel.updateReps(squat.id, 0, "")
        assertEquals(SetEntry(), setsOf(squat).single())

        viewModel.updateReps(squat.id, 0, "5")
        viewModel.toggleSetCompleted(squat.id, 0)
        viewModel.updateReps(squat.id, 0, "0")
        assertEquals(SetEntry(reps = "0"), setsOf(squat).single())
    }

    @Test
    fun `adding a set after a completed one starts uncompleted`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "5")
        viewModel.toggleSetCompleted(squat.id, 0)

        viewModel.addSet(squat.id)

        assertEquals(listOf(SetEntry(reps = "5", completed = true), SetEntry(reps = "5", id = 1)), setsOf(squat))
    }

    @Test
    fun `an exercise is completed only when it has sets and all are completed`() {
        assertFalse(WorkoutExerciseEntry(squat, emptyList()).allSetsCompleted)
        assertFalse(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5", completed = true), SetEntry(reps = "5"))).allSetsCompleted)
        assertTrue(WorkoutExerciseEntry(squat, listOf(SetEntry(reps = "5", completed = true))).allSetsCompleted)
    }

    @Test
    fun `set changes only affect the given exercise`() {
        viewModel.addExercise(squat)
        viewModel.addExercise(benchPress)

        viewModel.updateWeight(benchPress.id, 0, "60")
        viewModel.updateReps(benchPress.id, 0, "8")
        viewModel.addSet(benchPress.id)

        assertEquals(listOf(SetEntry()), setsOf(squat))
        assertEquals(listOf(SetEntry("60", "8"), SetEntry("60", "8", id = 1)), setsOf(benchPress))
    }

    @Test
    fun `weight accepts whole and decimal kg with a point or a comma`() {
        viewModel.addExercise(squat)

        for (weight in listOf("", "0", "100", "62.5", "62,5", "62,", "1.25", "1000")) {
            viewModel.updateWeight(squat.id, 0, weight)
            assertEquals(weight, setsOf(squat).single().weightKg)
        }
    }

    @Test
    fun `weight that isn't a kg value is ignored`() {
        viewModel.addExercise(squat)
        viewModel.updateWeight(squat.id, 0, "60")

        for (weight in listOf("abc", "-5", "1.255", "1,2,3", "6 0", "10000")) {
            viewModel.updateWeight(squat.id, 0, weight)
            assertEquals("60", setsOf(squat).single().weightKg)
        }
    }

    @Test
    fun `reps accept up to three digits only`() {
        viewModel.addExercise(squat)
        viewModel.updateReps(squat.id, 0, "12")

        for (reps in listOf("1.5", "-1", "abc", "1000")) {
            viewModel.updateReps(squat.id, 0, reps)
            assertEquals("12", setsOf(squat).single().reps)
        }

        viewModel.updateReps(squat.id, 0, "")
        assertEquals("", setsOf(squat).single().reps)
    }

    private fun workoutExercises() = viewModel.uiState.workoutExercises.map { it.exercise }

    private fun setsOf(exercise: Exercise) =
        viewModel.uiState.workoutExercises.single { it.exercise == exercise }.sets

    private fun assertErrorFor(result: RefreshResult, expectedMessage: Int) {
        viewModel
        exercisesRepository.refreshResult.complete(result)

        assertEquals(WorkoutUiState(exercisesErrorMessage = expectedMessage), viewModel.uiState)
    }
}
