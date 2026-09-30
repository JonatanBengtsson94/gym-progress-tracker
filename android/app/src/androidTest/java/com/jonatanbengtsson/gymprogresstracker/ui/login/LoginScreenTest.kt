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
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/** Exercises LoginScreen wired to a real LoginViewModel, with only the network faked. */
@RunWith(AndroidJUnit4::class)
class LoginScreenTest {

    @get:Rule
    val composeRule = createComposeRule()

    private class FakeAuthApi(private val result: LoginResult) : AuthApi {
        override suspend fun login(username: String, password: String) = result
    }

    private var loggedInSessionId: String? = null

    private fun setContent(result: LoginResult) {
        val viewModel = LoginViewModel(FakeAuthApi(result))
        composeRule.setContent {
            GymProgressTrackerTheme {
                LoginScreen(onLoggedIn = { loggedInSessionId = it }, viewModel = viewModel)
            }
        }
    }

    private fun str(id: Int) = InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun logIn() {
        composeRule.onNodeWithText(str(R.string.login_username)).performTextInput("alice")
        composeRule.onNodeWithText(str(R.string.login_password)).performTextInput("pw")
        composeRule.onNodeWithText(str(R.string.login_button)).performClick()
        composeRule.waitForIdle()
    }

    @Test
    fun successfulLoginReportsSessionId() {
        setContent(LoginResult.Success("session-123"))

        logIn()

        assertEquals("session-123", loggedInSessionId)
    }

    @Test
    fun invalidCredentialsShowsErrorAndStaysOnScreen() {
        setContent(LoginResult.InvalidCredentials)

        logIn()

        composeRule.onNodeWithText(str(R.string.login_error_invalid_credentials)).assertIsDisplayed()
        assertNull(loggedInSessionId)
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
