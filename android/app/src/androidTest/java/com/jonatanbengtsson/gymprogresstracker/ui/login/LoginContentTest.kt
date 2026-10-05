package com.jonatanbengtsson.gymprogresstracker.ui.login

import androidx.annotation.StringRes
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assert
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.assertTextContains
import androidx.compose.ui.test.hasProgressBarRangeInfo
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class LoginContentTest {

    @get:Rule
    val composeRule = createComposeRule()

    private val submitted = mutableListOf<Pair<String, String>>()

    private fun setContent(ownerUsername: String? = null, isLoading: Boolean = false, @StringRes errorMessage: Int? = null) {
        composeRule.setContent {
            GymProgressTrackerTheme {
                LoginContent(
                    ownerUsername = ownerUsername,
                    isLoading = isLoading,
                    errorMessage = errorMessage,
                    onLogin = { username, password -> submitted += username to password }
                )
            }
        }
    }

    private fun str(@StringRes id: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun usernameField() = composeRule.onNodeWithText(str(R.string.login_username))
    private fun passwordField() = composeRule.onNodeWithText(str(R.string.login_password))
    private fun loginButton() =
        composeRule.onNode(SemanticsMatcher.expectValue(SemanticsProperties.Role, Role.Button))

    @Test
    fun loginButtonIsDisabledUntilBothFieldsAreFilled() {
        setContent()
        loginButton().assertIsNotEnabled()

        usernameField().performTextInput("alice")
        loginButton().assertIsNotEnabled()

        passwordField().performTextInput("pw")
        loginButton().assertIsEnabled()
    }

    @Test
    fun theOwnersUsernameIsFilledInAndCantBeChanged() {
        setContent(ownerUsername = "alice")

        usernameField().assertTextContains("alice").assertIsNotEnabled()
        loginButton().assertIsNotEnabled()

        passwordField().performTextInput("pw")
        loginButton().performClick()

        assertEquals(listOf("alice" to "pw"), submitted)
    }

    @Test
    fun blankUsernameDoesNotEnableLogin() {
        setContent()

        usernameField().performTextInput("   ")
        passwordField().performTextInput("pw")

        loginButton().assertIsNotEnabled()
    }

    @Test
    fun clickingLoginSubmitsTrimmedUsernameAndUntouchedPassword() {
        setContent()

        usernameField().performTextInput("  alice  ")
        passwordField().performTextInput(" pw ")
        loginButton().performClick()

        assertEquals(listOf("alice" to " pw "), submitted)
    }

    @Test
    fun imeDoneOnPasswordSubmits() {
        setContent()

        usernameField().performTextInput("alice")
        passwordField().performTextInput("pw")
        passwordField().performImeAction()

        assertEquals(listOf("alice" to "pw"), submitted)
    }

    @Test
    fun imeDoneWithEmptyFieldsDoesNotSubmit() {
        setContent()

        passwordField().performImeAction()

        assertEquals(emptyList<Pair<String, String>>(), submitted)
    }

    @Test
    fun passwordIsMasked() {
        setContent()

        passwordField().assert(SemanticsMatcher.keyIsDefined(SemanticsProperties.Password))
    }

    @Test
    fun errorMessageIsShown() {
        setContent(errorMessage = R.string.login_error_invalid_credentials)

        composeRule.onNodeWithText(str(R.string.login_error_invalid_credentials)).assertIsDisplayed()
    }

    @Test
    fun noErrorMessageByDefault() {
        setContent()

        composeRule.onNodeWithText(str(R.string.login_error_invalid_credentials)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.login_error_network)).assertDoesNotExist()
        composeRule.onNodeWithText(str(R.string.login_error_server)).assertDoesNotExist()
    }

    @Test
    fun loadingDisablesInputAndShowsProgress() {
        setContent(isLoading = true)

        usernameField().assertIsNotEnabled()
        passwordField().assertIsNotEnabled()
        loginButton().assertIsNotEnabled()
        composeRule.onNodeWithText(str(R.string.login_button)).assertDoesNotExist()
        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertIsDisplayed()
    }
}
