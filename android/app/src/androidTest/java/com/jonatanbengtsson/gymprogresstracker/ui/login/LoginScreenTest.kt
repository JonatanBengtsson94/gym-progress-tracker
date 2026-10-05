package com.jonatanbengtsson.gymprogresstracker.ui.login

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.AuthApi
import com.jonatanbengtsson.gymprogresstracker.data.LoginResult
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.SyncRepository
import com.jonatanbengtsson.gymprogresstracker.data.SyncResult
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import kotlinx.coroutines.flow.MutableStateFlow
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/** Exercises LoginScreen wired to a real LoginViewModel, with only the network faked. */
@RunWith(AndroidJUnit4::class)
class LoginScreenTest {

    @get:Rule
    val composeRule = createComposeRule()

    private class FakeAuthApi(private val result: LoginResult) : AuthApi {
        val usernames = mutableListOf<String>()

        override suspend fun login(username: String, password: String): LoginResult {
            usernames += username
            return result
        }
    }

    private class FakeSessionRepository(session: SessionState) : SessionRepository {
        override val session = MutableStateFlow(session)

        override suspend fun logIn(username: String, sessionId: String) {
            session.value = SessionState.LoggedIn(sessionId, username)
        }

        override suspend fun endSession(sessionId: String) = error("Not used when logging in")
    }

    private class FakeSyncRepository : SyncRepository {
        var syncs = 0

        override suspend fun sync(): SyncResult {
            syncs++
            return SyncResult.Success
        }
    }

    private val syncRepository = FakeSyncRepository()
    private lateinit var authApi: FakeAuthApi
    private lateinit var sessionRepository: FakeSessionRepository
    private var loggedInCalls = 0

    private fun setContent(result: LoginResult, session: SessionState = SessionState.LoggedOut(null)) {
        authApi = FakeAuthApi(result)
        sessionRepository = FakeSessionRepository(session)
        val viewModel = LoginViewModel(authApi, sessionRepository, syncRepository)
        composeRule.setContent {
            GymProgressTrackerTheme {
                LoginScreen(onLoggedIn = { loggedInCalls++ }, viewModel = viewModel)
            }
        }
    }

    private fun str(id: Int) = InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun logIn(username: String? = "alice") {
        if (username != null) composeRule.onNodeWithText(str(R.string.login_username)).performTextInput(username)
        composeRule.onNodeWithText(str(R.string.login_password)).performTextInput("pw")
        composeRule.onNodeWithText(str(R.string.login_button)).performClick()
        composeRule.waitForIdle()
    }

    @Test
    fun theFirstLoginStartsTheSessionDownloadsAndLeaves() {
        setContent(LoginResult.Success("session-123"))

        logIn()

        assertEquals(SessionState.LoggedIn("session-123", "alice"), sessionRepository.session.value)
        assertEquals(1, syncRepository.syncs)
        assertEquals(1, loggedInCalls)
    }

    @Test
    fun loggingInAgainLogsInAsTheOwnerWithoutDownloading() {
        setContent(LoginResult.Success("session-123"), session = SessionState.LoggedOut("alice"))

        composeRule.onNodeWithText("alice").assertIsDisplayed()
        logIn(username = null)

        assertEquals(listOf("alice"), authApi.usernames)
        assertEquals(SessionState.LoggedIn("session-123", "alice"), sessionRepository.session.value)
        assertEquals(0, syncRepository.syncs)
        assertEquals(1, loggedInCalls)
    }

    @Test
    fun invalidCredentialsShowsErrorAndStaysOnScreen() {
        setContent(LoginResult.InvalidCredentials)

        logIn()

        composeRule.onNodeWithText(str(R.string.login_error_invalid_credentials)).assertIsDisplayed()
        assertEquals(SessionState.LoggedOut(null), sessionRepository.session.value)
        assertEquals(0, loggedInCalls)
    }

    @Test
    fun networkErrorShowsError() {
        setContent(LoginResult.NetworkError)

        logIn()

        composeRule.onNodeWithText(str(R.string.login_error_network)).assertIsDisplayed()
    }

    @Test
    fun serverErrorShowsError() {
        setContent(LoginResult.ServerError)

        logIn()

        composeRule.onNodeWithText(str(R.string.login_error_server)).assertIsDisplayed()
    }
}
