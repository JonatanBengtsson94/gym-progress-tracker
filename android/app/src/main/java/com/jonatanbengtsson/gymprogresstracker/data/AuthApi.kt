package com.jonatanbengtsson.gymprogresstracker.data

import org.json.JSONObject

sealed interface LoginResult {
    data class Success(val sessionId: String) : LoginResult
    data object InvalidCredentials : LoginResult
    data object NetworkError : LoginResult
    data object ServerError : LoginResult
}

interface AuthApi {
    suspend fun login(username: String, password: String): LoginResult
}

class HttpAuthApi(private val client: ApiClient) : AuthApi {

    override suspend fun login(username: String, password: String): LoginResult {
        val body = JSONObject()
            .put("username", username)
            .put("password", password)

        return when (val result = client.sendWithoutSession("POST", "/login", body) { it.getString("session_id") }) {
            is ApiResult.Success -> LoginResult.Success(result.value)
            ApiResult.Unauthorized -> LoginResult.InvalidCredentials
            ApiResult.NetworkError -> LoginResult.NetworkError
            ApiResult.ServerError -> LoginResult.ServerError
        }
    }
}
