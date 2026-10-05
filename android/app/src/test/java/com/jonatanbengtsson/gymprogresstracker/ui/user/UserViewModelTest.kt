package com.jonatanbengtsson.gymprogresstracker.ui.user

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.FakeSessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeSyncRepository
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
    private val syncRepository = FakeSyncRepository()

    // Created lazily so each test can set up the fakes first.
    private val viewModel by lazy { UserViewModel(sessionRepository, workoutsRepository, syncRepository) }

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

        assertEquals(1, syncRepository.syncs)
        assertTrue(viewModel.uiState.isSyncing)

        syncRepository.syncResult.complete(SyncResult.Success)

        assertFalse(viewModel.uiState.isSyncing)
        assertNull(viewModel.uiState.syncErrorMessage)
    }

    @Test
    fun `syncing is ignored while a sync is in flight`() {
        viewModel.sync()

        viewModel.sync()

        assertEquals(1, syncRepository.syncs)
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
        syncRepository.syncResult.complete(SyncResult.NetworkError)
        syncRepository.syncResult = CompletableDeferred()

        viewModel.sync()

        assertNull(viewModel.uiState.syncErrorMessage)
        assertEquals(2, syncRepository.syncs)
    }

    @Test
    fun `syncing without a session asks to log in instead`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        viewModel.sync()

        assertEquals(0, syncRepository.syncs)
        assertTrue(viewModel.uiState.logInRequested)
    }

    @Test
    fun `a session the server ended asks to log in without an error`() {
        viewModel.sync()
        syncRepository.syncResult.complete(SyncResult.NotLoggedIn)

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

        assertEquals(1, syncRepository.syncs)
        assertTrue(viewModel.uiState.isSyncing)
    }

    @Test
    fun `syncs again once the user has logged in after the server ended the session`() {
        viewModel.sync()
        syncRepository.syncResult.complete(SyncResult.NotLoggedIn)
        syncRepository.syncResult = CompletableDeferred()
        viewModel.onLogInShown()
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        sessionRepository.session.value = SessionState.LoggedIn("session-456", "alice")

        assertEquals(2, syncRepository.syncs)
        assertTrue(viewModel.uiState.isSyncing)
    }

    @Test
    fun `a sync waiting for a login still runs if the login is left and done later`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")
        viewModel.sync()
        viewModel.onLogInShown()

        sessionRepository.session.value = SessionState.LoggedOut("alice")
        sessionRepository.session.value = SessionState.LoggedIn("session-123", "alice")

        assertEquals(1, syncRepository.syncs)
    }

    @Test
    fun `shows when the session ends`() {
        viewModel

        sessionRepository.session.value = SessionState.LoggedOut("alice")

        assertEquals(UserUiState(username = "alice"), viewModel.uiState)
    }

    @Test
    fun `logging in without a sync waiting doesn't sync`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")
        viewModel

        sessionRepository.session.value = SessionState.LoggedIn("session-123", "alice")

        assertEquals(0, syncRepository.syncs)
    }

    private fun assertSyncErrorFor(result: SyncResult, expectedMessage: Int) {
        viewModel.sync()
        syncRepository.syncResult.complete(result)

        assertEquals(expectedMessage, viewModel.uiState.syncErrorMessage)
        assertFalse(viewModel.uiState.logInRequested)
    }
}
