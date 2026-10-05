package com.jonatanbengtsson.gymprogresstracker.ui.user

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.FakeSessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeWorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.SyncResult
import com.jonatanbengtsson.gymprogresstracker.ui.login.MainDispatcherRule
import kotlinx.coroutines.CompletableDeferred
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class UserViewModelTest {

    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val sessionRepository = FakeSessionRepository(SessionState.LoggedIn("session-123", "alice"))
    private val workoutsRepository = FakeWorkoutsRepository()

    // Created lazily so each test can set up the fakes first.
    private val viewModel by lazy { UserViewModel(sessionRepository, workoutsRepository) }

    @Test
    fun `shows who is logged in`() {
        assertEquals(UserUiState(username = "alice", isLoggedIn = true), viewModel.uiState)
    }

    @Test
    fun `shows who the device belongs to while logged out`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        assertEquals(UserUiState(username = "alice"), viewModel.uiState)
    }

    @Test
    fun `shows how many saved workouts are waiting to sync`() {
        viewModel

        workoutsRepository.pendingCount.value = 2

        assertEquals(2, viewModel.uiState.pendingWorkouts)
    }

    @Test
    fun `syncing shows progress until it's done`() {
        viewModel.sync()

        assertEquals(1, workoutsRepository.syncs)
        assertTrue(viewModel.uiState.isSyncing)

        workoutsRepository.syncResult.complete(SyncResult.Success)

        assertFalse(viewModel.uiState.isSyncing)
        assertNull(viewModel.uiState.syncErrorMessage)
    }

    @Test
    fun `syncing is ignored while a sync is in flight`() {
        viewModel.sync()

        viewModel.sync()

        assertEquals(1, workoutsRepository.syncs)
    }

    @Test
    fun `network error syncing shows network error`() {
        assertSyncErrorFor(SyncResult.NetworkError, R.string.user_sync_error_network)
    }

    @Test
    fun `server error syncing shows server error`() {
        assertSyncErrorFor(SyncResult.ServerError, R.string.user_sync_error_server)
    }

    @Test
    fun `syncing again clears the last error`() {
        viewModel.sync()
        workoutsRepository.syncResult.complete(SyncResult.NetworkError)
        workoutsRepository.syncResult = CompletableDeferred()

        viewModel.sync()

        assertNull(viewModel.uiState.syncErrorMessage)
        assertEquals(2, workoutsRepository.syncs)
    }

    @Test
    fun `syncing without a session asks to log in instead`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        viewModel.sync()

        assertEquals(0, workoutsRepository.syncs)
        assertTrue(viewModel.uiState.logInRequested)
    }

    @Test
    fun `a session the server ended asks to log in without an error`() {
        viewModel.sync()
        workoutsRepository.syncResult.complete(SyncResult.NotLoggedIn)

        assertTrue(viewModel.uiState.logInRequested)
        assertNull(viewModel.uiState.syncErrorMessage)
    }

    @Test
    fun `showing the login clears the request`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")
        viewModel.sync()

        viewModel.onLogInShown()

        assertFalse(viewModel.uiState.logInRequested)
    }

    @Test
    fun `syncs once the user has logged in`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")
        viewModel.sync()
        viewModel.onLogInShown()

        sessionRepository.session.value = SessionState.LoggedIn("session-123", "alice")

        assertEquals(1, workoutsRepository.syncs)
        assertTrue(viewModel.uiState.isSyncing)
    }

    @Test
    fun `logging in without a sync waiting doesn't sync`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")
        viewModel

        sessionRepository.session.value = SessionState.LoggedIn("session-123", "alice")

        assertEquals(0, workoutsRepository.syncs)
    }

    private fun assertSyncErrorFor(result: SyncResult, expectedMessage: Int) {
        viewModel.sync()
        workoutsRepository.syncResult.complete(result)

        assertEquals(expectedMessage, viewModel.uiState.syncErrorMessage)
        assertFalse(viewModel.uiState.logInRequested)
    }
}
