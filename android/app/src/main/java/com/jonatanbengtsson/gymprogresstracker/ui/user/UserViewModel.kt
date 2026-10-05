package com.jonatanbengtsson.gymprogresstracker.ui.user

import androidx.annotation.StringRes
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.appContainer
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.SessionState
import com.jonatanbengtsson.gymprogresstracker.data.SyncRepository
import com.jonatanbengtsson.gymprogresstracker.data.SyncResult
import com.jonatanbengtsson.gymprogresstracker.data.WorkoutsRepository
import com.jonatanbengtsson.gymprogresstracker.data.owner
import kotlinx.coroutines.launch

data class UserUiState(
    /** Who the device belongs to, or null if nobody has logged in on it yet. */
    val username: String? = null,
    val isLoggedIn: Boolean = false,
    /** How many saved workouts haven't been synced to the server yet. */
    val pendingWorkouts: Int = 0,
    /** True while the saved workouts are being synced. */
    val isSyncing: Boolean = false,
    /** Why syncing failed last time. */
    @StringRes val syncErrorMessage: Int? = null,
    /** True when syncing needs the user to log in first. Cleared by [UserViewModel.onLogInShown]. */
    val logInRequested: Boolean = false
)

class UserViewModel(
    private val sessionRepository: SessionRepository,
    private val workoutsRepository: WorkoutsRepository,
    private val syncRepository: SyncRepository
) : ViewModel() {

    var uiState by mutableStateOf(UserUiState())
        private set

    /** True when a sync is waiting for the user to log in. */
    private var syncAfterLogIn = false

    init {
        viewModelScope.launch {
            sessionRepository.session.collect { session ->
                val isLoggedIn = session is SessionState.LoggedIn
                uiState = uiState.copy(username = session.owner, isLoggedIn = isLoggedIn)
                if (isLoggedIn && syncAfterLogIn) sync()
            }
        }
        viewModelScope.launch {
            workoutsRepository.pendingCount.collect { count -> uiState = uiState.copy(pendingWorkouts = count) }
        }
    }

    /**
     * Sends the saved workouts to the server and downloads the templates and exercises. Without a
     * session, asks the user to log in and syncs once they have.
     */
    fun sync() {
        if (uiState.isSyncing) return
        if (!uiState.isLoggedIn) {
            requestLogIn()
            return
        }
        syncAfterLogIn = false
        uiState = uiState.copy(isSyncing = true, syncErrorMessage = null)

        viewModelScope.launch {
            val result = syncRepository.sync()
            uiState = uiState.copy(
                isSyncing = false,
                syncErrorMessage = when (result) {
                    SyncResult.NetworkError -> R.string.user_sync_error_network
                    SyncResult.ServerError -> R.string.user_sync_error_server
                    SyncResult.Success, SyncResult.NotLoggedIn -> null
                }
            )
            if (result == SyncResult.NotLoggedIn) requestLogIn()
        }
    }

    fun onLogInShown() {
        uiState = uiState.copy(logInRequested = false)
    }

    private fun requestLogIn() {
        syncAfterLogIn = true
        uiState = uiState.copy(logInRequested = true)
    }

    companion object {
        val Factory = viewModelFactory {
            initializer { UserViewModel(appContainer.sessionRepository, appContainer.workoutsRepository, appContainer.syncRepository) }
        }
    }
}
