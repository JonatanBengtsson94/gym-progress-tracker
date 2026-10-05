package com.jonatanbengtsson.gymprogresstracker.ui.user

import androidx.annotation.StringRes
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasProgressBarRangeInfo
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class UserContentTest {

    @get:Rule
    val composeRule = createComposeRule()

    private var syncClicks = 0

    private fun setContent(uiState: UserUiState) {
        composeRule.setContent {
            GymProgressTrackerTheme {
                UserContent(uiState = uiState, onSync = { syncClicks++ })
            }
        }
    }

    private fun str(@StringRes id: Int) = InstrumentationRegistry.getInstrumentation().targetContext.getString(id)

    private fun waitingToSync(count: Int) =
        InstrumentationRegistry.getInstrumentation().targetContext.resources.getQuantityString(R.plurals.user_pending, count, count)

    @Test
    fun showsWhoIsLoggedIn() {
        setContent(UserUiState(username = "alice", isLoggedIn = true))

        composeRule.onNodeWithText(str(R.string.user_logged_in_as)).assertIsDisplayed()
        composeRule.onNodeWithText("alice").assertIsDisplayed()
    }

    @Test
    fun showsWhoTheDeviceBelongsToWhileLoggedOut() {
        setContent(UserUiState(username = "alice"))

        composeRule.onNodeWithText(str(R.string.user_logged_out)).assertIsDisplayed()
        composeRule.onNodeWithText("alice").assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.user_logged_in_as)).assertDoesNotExist()
    }

    @Test
    fun workoutsWaitingToSyncShowHowMany() {
        setContent(UserUiState(username = "alice", pendingWorkouts = 2))

        composeRule.onNodeWithText(waitingToSync(2)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.user_all_synced)).assertDoesNotExist()
    }

    @Test
    fun nothingWaitingSaysEverythingIsSyncedAndCanStillSync() {
        setContent(UserUiState(username = "alice"))

        composeRule.onNodeWithText(str(R.string.user_all_synced)).assertIsDisplayed()
        composeRule.onNodeWithText(str(R.string.user_sync)).performClick()

        assertEquals(1, syncClicks)
    }

    @Test
    fun syncingShowsProgressAndCantBePressedAgain() {
        setContent(UserUiState(username = "alice", pendingWorkouts = 1, isSyncing = true))

        composeRule.onNodeWithText(str(R.string.user_sync)).assertDoesNotExist()
        composeRule.onNode(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).assertIsDisplayed()
        composeRule.onNode(SemanticsMatcher.expectValue(SemanticsProperties.Role, Role.Button)).assertIsNotEnabled()
    }

    @Test
    fun aFailedSyncShowsWhy() {
        setContent(UserUiState(username = "alice", pendingWorkouts = 1, syncErrorMessage = R.string.user_sync_error_network))

        composeRule.onNodeWithText(str(R.string.user_sync_error_network)).assertIsDisplayed()
    }
}
