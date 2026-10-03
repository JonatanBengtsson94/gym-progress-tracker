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
import com.jonatanbengtsson.gymprogresstracker.data.SessionRepository
import kotlinx.coroutines.launch

data class LoginUiState(
    val isLoading: Boolean = false,
    @StringRes val errorMessage: Int? = null
)

class LoginViewModel(
    private val authApi: AuthApi,
    private val sessionRepository: SessionRepository
) : ViewModel() {

    var uiState by mutableStateOf(LoginUiState())
        private set

    fun login(username: String, password: String) {
        if (uiState.isLoading) return
        uiState = LoginUiState(isLoading = true)

        viewModelScope.launch {
            when (val result = authApi.login(username, password)) {
                // Stays loading; the app leaves this screen once the session is saved.
                is LoginResult.Success -> sessionRepository.logIn(username, result.sessionId)
                LoginResult.InvalidCredentials -> uiState = LoginUiState(errorMessage = R.string.login_error_invalid_credentials)
                LoginResult.NetworkError -> uiState = LoginUiState(errorMessage = R.string.login_error_network)
                LoginResult.ServerError -> uiState = LoginUiState(errorMessage = R.string.login_error_server)
            }
        }
    }
}
