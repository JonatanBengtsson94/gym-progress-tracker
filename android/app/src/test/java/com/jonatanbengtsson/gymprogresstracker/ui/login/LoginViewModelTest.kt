package com.jonatanbengtsson.gymprogresstracker.ui.login

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.AuthApi
import com.jonatanbengtsson.gymprogresstracker.data.FakeSessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.FakeSyncRepository
import com.jonatanbengtsson.gymprogresstracker.data.LoginResult
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.SyncResult
import kotlinx.coroutines.CompletableDeferred
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

class LoginViewModelTest {

    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    /** Suspends each login until the test completes [response]. */
    private class FakeAuthApi : AuthApi {
        var response = CompletableDeferred<LoginResult>()
        val calls = mutableListOf<Pair<String, String>>()

        override suspend fun login(username: String, password: String): LoginResult {
            calls += username to password
            return response.await()
        }
    }

    private val authApi = FakeAuthApi()
    private val sessionRepository = FakeSessionRepository()
    private val syncRepository = FakeSyncRepository()

    // Created lazily so a test can give the device an owner first.
    private val viewModel by lazy { LoginViewModel(authApi, sessionRepository, syncRepository) }

    @Test
    fun `initial state is idle`() {
        assertEquals(LoginUiState(), viewModel.uiState)
    }

    @Test
    fun `login forwards credentials to the api`() {
        viewModel.login("alice", "pw")

        assertEquals(listOf("alice" to "pw"), authApi.calls)
    }

    @Test
    fun `state is loading while the request is in flight`() {
        viewModel.login("alice", "pw")

        assertEquals(LoginUiState(isLoading = true), viewModel.uiState)
    }

    @Test
    fun `the first login downloads the user's data before it's done`() {
        viewModel.login("alice", "pw")
        authApi.response.complete(LoginResult.Success("session-123"))

        assertEquals(listOf("alice" to "session-123"), sessionRepository.logIns)
        assertEquals(SessionState.LoggedIn("session-123", "alice"), sessionRepository.session.value)
        assertEquals(1, syncRepository.syncs)
        assertEquals(LoginUiState(isLoading = true), viewModel.uiState)

        syncRepository.syncResult.complete(SyncResult.Success)

        assertEquals(LoginUiState(isLoggedIn = true), viewModel.uiState)
    }

    @Test
    fun `the first login is done even if downloading fails`() {
        viewModel.login("alice", "pw")
        authApi.response.complete(LoginResult.Success("session-123"))

        syncRepository.syncResult.complete(SyncResult.NetworkError)

        assertEquals(LoginUiState(isLoggedIn = true), viewModel.uiState)
    }

    @Test
    fun `logging in again doesn't download anything`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        viewModel.login("alice", "pw")
        authApi.response.complete(LoginResult.Success("session-123"))

        assertEquals(0, syncRepository.syncs)
        assertEquals(LoginUiState(ownerUsername = "alice", isLoggedIn = true), viewModel.uiState)
    }

    @Test
    fun `login is ignored once logged in`() {
        viewModel.login("alice", "pw")
        authApi.response.complete(LoginResult.Success("session-123"))
        syncRepository.syncResult.complete(SyncResult.Success)

        viewModel.login("alice", "pw")

        assertEquals(1, authApi.calls.size)
    }

    @Test
    fun `a device that belongs to someone shows who`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        assertEquals(LoginUiState(ownerUsername = "alice"), viewModel.uiState)
    }

    @Test
    fun `a device that belongs to someone only logs in as them`() {
        sessionRepository.session.value = SessionState.LoggedOut("alice")

        viewModel.login("bob", "pw")
        authApi.response.complete(LoginResult.Success("session-123"))

        assertEquals(listOf("alice" to "pw"), authApi.calls)
        assertEquals(SessionState.LoggedIn("session-123", "alice"), sessionRepository.session.value)
    }

    @Test
    fun `invalid credentials shows invalid credentials error`() {
        assertErrorFor(LoginResult.InvalidCredentials, R.string.login_error_invalid_credentials)
    }

    @Test
    fun `network error shows network error`() {
        assertErrorFor(LoginResult.NetworkError, R.string.login_error_network)
    }

    @Test
    fun `server error shows server error`() {
        assertErrorFor(LoginResult.ServerError, R.string.login_error_server)
    }

    @Test
    fun `login is ignored while a request is in flight`() {
        viewModel.login("alice", "pw")
        viewModel.login("bob", "other")

        assertEquals(listOf("alice" to "pw"), authApi.calls)
    }

    @Test
    fun `retrying after an error clears the error`() {
        viewModel.login("alice", "wrong")
        authApi.response.complete(LoginResult.InvalidCredentials)
        authApi.response = CompletableDeferred()

        viewModel.login("alice", "pw")

        assertEquals(LoginUiState(isLoading = true), viewModel.uiState)
        authApi.response.complete(LoginResult.Success("session-123"))
        syncRepository.syncResult.complete(SyncResult.Success)
        assertEquals(LoginUiState(isLoggedIn = true), viewModel.uiState)
        assertEquals(SessionState.LoggedIn("session-123", "alice"), sessionRepository.session.value)
    }

    private fun assertErrorFor(result: LoginResult, expectedMessage: Int) {
        viewModel.login("alice", "pw")
        authApi.response.complete(result)

        assertEquals(LoginUiState(errorMessage = expectedMessage), viewModel.uiState)
    }
}
