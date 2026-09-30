package com.jonatanbengtsson.gymprogresstracker.ui.login

import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.AuthApi
import com.jonatanbengtsson.gymprogresstracker.data.LoginResult
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
    private val viewModel = LoginViewModel(authApi)

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
    fun `success exposes the session id`() {
        viewModel.login("alice", "pw")
        authApi.response.complete(LoginResult.Success("session-123"))

        assertEquals(LoginUiState(sessionId = "session-123"), viewModel.uiState)
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
        assertEquals(LoginUiState(sessionId = "session-123"), viewModel.uiState)
    }

    private fun assertErrorFor(result: LoginResult, expectedMessage: Int) {
        viewModel.login("alice", "pw")
        authApi.response.complete(result)

        assertEquals(LoginUiState(errorMessage = expectedMessage), viewModel.uiState)
    }
}
