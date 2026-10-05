package com.jonatanbengtsson.gymprogresstracker.ui.login

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
import com.jonatanbengtsson.gymprogresstracker.data.AuthApi
import com.jonatanbengtsson.gymprogresstracker.data.LoginResult
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import com.jonatanbengtsson.gymprogresstracker.data.SyncRepository
import com.jonatanbengtsson.gymprogresstracker.data.owner
import kotlinx.coroutines.launch

data class LoginUiState(
    /** Who the device belongs to, the only user who can log in on it, or null if nobody has yet. */
    val ownerUsername: String? = null,
    /** True while logging in, and on the device's first login while the user's data is downloaded. */
    val isLoading: Boolean = false,
    @StringRes val errorMessage: Int? = null,
    /** True once the session has been saved. */
    val isLoggedIn: Boolean = false
)

class LoginViewModel(
    private val authApi: AuthApi,
    private val sessionRepository: SessionRepository,
    private val syncRepository: SyncRepository
) : ViewModel() {

    var uiState by mutableStateOf(LoginUiState(ownerUsername = sessionRepository.session.value.owner))
        private set

    /**
     * Logs in as [username], or as the device's owner if it has one. The first login on the device also
     * downloads the user's templates and exercises. If that fails, they can be synced later.
     */
    fun login(username: String, password: String) {
        if (uiState.isLoading || uiState.isLoggedIn) return
        val loginUsername = uiState.ownerUsername ?: username
        uiState = uiState.copy(isLoading = true, errorMessage = null)

        viewModelScope.launch {
            val result = authApi.login(loginUsername, password)
            if (result is LoginResult.Success) {
                sessionRepository.logIn(loginUsername, result.sessionId)
                if (uiState.ownerUsername == null) syncRepository.sync()
            }
            uiState = uiState.copy(
                isLoading = false,
                isLoggedIn = result is LoginResult.Success,
                errorMessage = when (result) {
                    is LoginResult.Success -> null
                    LoginResult.InvalidCredentials -> R.string.login_error_invalid_credentials
                    LoginResult.NetworkError -> R.string.login_error_network
                    LoginResult.ServerError -> R.string.login_error_server
                }
            )
        }
    }

    companion object {
        val Factory = viewModelFactory {
            initializer { LoginViewModel(appContainer.authApi, appContainer.sessionRepository, appContainer.syncRepository) }
        }
    }
}
