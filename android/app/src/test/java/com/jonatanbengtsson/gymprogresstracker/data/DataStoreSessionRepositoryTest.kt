package com.jonatanbengtsson.gymprogresstracker.data

import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File
import java.time.Instant

class DataStoreSessionRepositoryTest {

    @get:Rule
    val folder = TemporaryFolder()

    private val squat = Exercise(testId(1), "Squat (Barbell)")
    private val workout = ActiveWorkout(Instant.parse("2026-10-04T17:00:00Z"), exercises = listOf(WorkoutExerciseEntry(squat)))
    private val activeWorkoutRepository = FakeActiveWorkoutRepository(workout)
    private val pendingWorkout = PendingWorkout(
        testId(20),
        testId(10),
        templateIsNew = true,
        FinishedWorkout("Leg day", Instant.parse("2026-10-03T17:00:00Z"), Instant.parse("2026-10-03T18:00:00Z"), emptyList())
    )
    private val pendingWorkoutsRepository = FakePendingWorkoutsRepository(listOf(pendingWorkout))

    private val file by lazy { File(folder.root, "session.preferences_pb") }

    /** A repository over the same file each time, as after the app restarts. Cancel [scope] before the next one. */
    private fun repository(scope: CoroutineScope) = DataStoreSessionRepository(
        PreferenceDataStoreFactory.create(scope = scope) { file },
        activeWorkoutRepository,
        pendingWorkoutsRepository,
        scope
    )

    private fun TestScope.newRunScope() = CoroutineScope(coroutineContext + Job())

    private suspend fun SessionRepository.loadedSession() = session.first { it != SessionState.Loading }

    @Test
    fun `nobody is logged in at first`() = runTest {
        val repository = repository(backgroundScope)

        assertEquals(SessionState.Loading, repository.session.value)
        assertEquals(SessionState.LoggedOut, repository.loadedSession())
    }

    @Test
    fun `a session survives a restart`() = runTest {
        val firstRun = newRunScope()
        repository(firstRun).logIn("alice", "session-123")
        firstRun.cancel()

        val repository = repository(backgroundScope)

        assertEquals(SessionState.LoggedIn("session-123"), repository.loadedSession())
    }

    @Test
    fun `ending the session logs out`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-123")

        repository.endSession("session-123")

        assertEquals(SessionState.LoggedOut, repository.session.first { it == SessionState.LoggedOut })
    }

    @Test
    fun `ending a session that has been replaced is ignored`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-1")
        repository.logIn("alice", "session-2")

        repository.endSession("session-1")

        assertEquals(SessionState.LoggedIn("session-2"), repository.session.first { it == SessionState.LoggedIn("session-2") })
    }

    @Test
    fun `logging out ends the current session`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-123")

        repository.logOut()

        assertEquals(SessionState.LoggedOut, repository.session.first { it == SessionState.LoggedOut })
    }

    @Test
    fun `logging out keeps the workouts for the same user's next login`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-1")
        activeWorkoutRepository.workout.value = workout
        pendingWorkoutsRepository.workouts.value = listOf(pendingWorkout)
        repository.logOut()

        repository.logIn("alice", "session-2")

        assertEquals(workout, activeWorkoutRepository.workout.value)
        assertEquals(listOf(pendingWorkout), pendingWorkoutsRepository.workouts.value)
    }

    @Test
    fun `the first login throws away workouts nobody owns`() = runTest {
        repository(backgroundScope).logIn("alice", "session-123")

        assertEquals(ActiveWorkout(), activeWorkoutRepository.workout.value)
        assertEquals(emptyList<PendingWorkout>(), pendingWorkoutsRepository.workouts.value)
    }

    @Test
    fun `the same user logging in again keeps their workout`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-1")
        activeWorkoutRepository.workout.value = workout
        repository.endSession("session-1")

        repository.logIn("alice", "session-2")

        assertEquals(workout, activeWorkoutRepository.workout.value)
    }

    @Test
    fun `someone else logging in throws away the workouts`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-1")
        activeWorkoutRepository.workout.value = workout
        pendingWorkoutsRepository.workouts.value = listOf(pendingWorkout)
        repository.endSession("session-1")

        repository.logIn("bob", "session-2")

        assertEquals(ActiveWorkout(), activeWorkoutRepository.workout.value)
        assertEquals(emptyList<PendingWorkout>(), pendingWorkoutsRepository.workouts.value)
    }

    @Test
    fun `logging in waits for the saved workout before throwing it away`() = runTest {
        activeWorkoutRepository.workout.value = null
        val repository = repository(backgroundScope)
        val loggingIn = backgroundScope.async { repository.logIn("alice", "session-123") }

        // In real time, since the file is read off the test's dispatcher.
        assertNull(withContext(Dispatchers.Default) { withTimeoutOrNull(1_000) { loggingIn.await() } })
        activeWorkoutRepository.workout.value = workout
        loggingIn.await()

        assertEquals(ActiveWorkout(), activeWorkoutRepository.workout.value)
    }
}
