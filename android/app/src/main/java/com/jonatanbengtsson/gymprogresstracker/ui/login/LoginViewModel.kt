package com.jonatanbengtsson.gymprogresstracker.ui.login

import androidx.annotation.StringRes
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.data.AuthApi
import com.jonatanbengtsson.gymprogresstracker.data.LoginResult
import kotlinx.coroutines.launch

data class LoginUiState(
    val isLoading: Boolean = false,
    @StringRes val errorMessage: Int? = null,
    val sessionId: String? = null
)

class LoginViewModel(private val authApi: AuthApi) : ViewModel() {

    var uiState by mutableStateOf(LoginUiState())
        private set

    fun login(username: String, password: String) {
        if (uiState.isLoading) return
        uiState = LoginUiState(isLoading = true)

        viewModelScope.launch {
            uiState = when (val result = authApi.login(username, password)) {
                is LoginResult.Success -> LoginUiState(sessionId = result.sessionId)
                LoginResult.InvalidCredentials -> LoginUiState(errorMessage = R.string.login_error_invalid_credentials)
                LoginResult.NetworkError -> LoginUiState(errorMessage = R.string.login_error_network)
                LoginResult.ServerError -> LoginUiState(errorMessage = R.string.login_error_server)
            }
        }
    }

    /**
     * Clears the session id once the screen has handed it on. The view model outlives the screen,
     * so without this a later return to login (e.g. after the session expires) would immediately
     * report the stale session again.
     */
    fun onLoggedInHandled() {
        uiState = LoginUiState()
    }
}
