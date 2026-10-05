package com.jonatanbengtsson.gymprogresstracker.ui.user

import androidx.annotation.StringRes
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.FinishedWorkout
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.SyncRepository
import com.jonatanbengtsson.gymprogresstracker.data.SyncResult
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import kotlinx.coroutines.flow.MutableStateFlow
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import kotlin.uuid.Uuid

/** Exercises UserScreen wired to a real UserViewModel, with the session and syncing faked. */
@RunWith(AndroidJUnit4::class)
class UserScreenTest {

    @get:Rule
    val composeRule = createComposeRule()

    private class FakeSessionRepository(session: SessionState) : SessionRepository {
        override val session = MutableStateFlow(session)

        override suspend fun logIn(username: String, sessionId: String) {
            session.value = SessionState.LoggedIn(sessionId, username)
        }

        override suspend fun endSession(sessionId: String) = error("Not used on the user screen")
    }

    private class FakeWorkoutsRepository : WorkoutsRepository {
        override val pendingCount = MutableStateFlow(1)
        override suspend fun save(workoutId: Uuid, workout: FinishedWorkout) = error("Not used on the user screen")
        override suspend fun send() = error("Not used on the user screen")
    }

    /** Answers each sync with the next of [results]. */
    private class FakeSyncRepository(vararg results: SyncResult) : SyncRepository {
        private val results = ArrayDeque(results.toList())
        var syncs = 0

        override suspend fun sync(): SyncResult {
            syncs++
            return results.removeFirst()
        }
    }

    private lateinit var sessionRepository: FakeSessionRepository
    private lateinit var syncRepository: FakeSyncRepository
    private var logInCalls = 0

    private fun setContent(session: SessionState, vararg syncResults: SyncResult) {
        sessionRepository = FakeSessionRepository(session)
        syncRepository = FakeSyncRepository(*syncResults)
        // On the main thread, like viewModel() would, since it updates its state as it starts.
        val viewModel = composeRule.runOnUiThread { UserViewModel(sessionRepository, FakeWorkoutsRepository(), syncRepository) }
        composeRule.setContent {
            GymProgressTrackerTheme {
                UserScreen(onLogIn = { logInCalls++ }, viewModel = viewModel)
            }
        }
    }

    private fun str(@StringRes id: Int) = InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun sync() {
        composeRule.onNodeWithText(str(R.string.user_sync)).performClick()
        composeRule.waitForIdle()
    }

    @Test
    fun syncingWhileLoggedInDoesNotOpenTheLogin() {
        setContent(SessionState.LoggedIn("session-123", "alice"), SyncResult.Success)

        sync()

        assertEquals(1, syncRepository.syncs)
        assertEquals(0, logInCalls)
    }

    @Test
    fun syncingWithoutASessionOpensTheLoginOnce() {
        setContent(SessionState.LoggedOut("alice"))

        sync()

        assertEquals(0, syncRepository.syncs)
        assertEquals(1, logInCalls)
    }

    @Test
    fun aSessionTheServerEndedOpensTheLogin() {
        setContent(SessionState.LoggedIn("session-123", "alice"), SyncResult.NotLoggedIn)

        sync()

        assertEquals(1, logInCalls)
    }

    @Test
    fun loggingInFinishesTheSync() {
        setContent(SessionState.LoggedOut("alice"), SyncResult.Success)
        sync()

        composeRule.runOnIdle { sessionRepository.session.value = SessionState.LoggedIn("session-123", "alice") }
        composeRule.waitForIdle()

        assertEquals(1, syncRepository.syncs)
        composeRule.onNodeWithText(str(R.string.user_logged_in_as)).assertIsDisplayed()
    }
}
