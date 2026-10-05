package com.jonatanbengtsson.gymprogresstracker.data

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.emptyPreferences
import androidx.datastore.preferences.core.stringPreferencesKey
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import java.io.IOException

sealed interface SessionState {
    /** The saved session hasn't been read from disk yet. */
    data object Loading : SessionState

    /** [username] is who the device belongs to, or null if nobody has logged in on it yet. */
    data class LoggedOut(val username: String?) : SessionState

    data class LoggedIn(val sessionId: String, val username: String) : SessionState
}

/** Who the device belongs to, or null if nobody has logged in on it yet or the session is still loading. */
val SessionState.owner: String?
    get() = when (this) {
        SessionState.Loading -> null
        is SessionState.LoggedOut -> username
        is SessionState.LoggedIn -> username
    }

/**
 * The session used to talk to the server, kept on the device so the user stays logged in across
 * restarts. The app works without one; only requests for the user's own data need it. The first
 * login makes that user the device's owner.
 */
interface SessionRepository {
    val session: StateFlow<SessionState>

    /** Starts [sessionId] for [username], who the device belongs to from then on. */
    suspend fun logIn(username: String, sessionId: String)

    /** Ends [sessionId] once the server rejects it. Ignored if another session has started since. */
    suspend fun endSession(sessionId: String)
}

class DataStoreSessionRepository(
    private val dataStore: DataStore<Preferences>,
    externalScope: CoroutineScope
) : SessionRepository {

    // An unreadable file is treated as empty, which logs the user out rather than crashing.
    private val preferences = dataStore.data.catch { if (it is IOException) emit(emptyPreferences()) else throw it }

    override val session: StateFlow<SessionState> = preferences
        .map { preferences ->
            val sessionId = preferences[SESSION_ID]
            val username = preferences[USERNAME]
            if (sessionId != null && username != null) SessionState.LoggedIn(sessionId, username) else SessionState.LoggedOut(username)
        }
        .stateIn(externalScope, SharingStarted.Eagerly, SessionState.Loading)

    override suspend fun logIn(username: String, sessionId: String) {
        dataStore.edit { preferences ->
            preferences[USERNAME] = username
            preferences[SESSION_ID] = sessionId
        }
    }

    override suspend fun endSession(sessionId: String) {
        dataStore.edit { preferences ->
            if (preferences[SESSION_ID] == sessionId) preferences.remove(SESSION_ID)
        }
    }

    private companion object {
        val SESSION_ID = stringPreferencesKey("session_id")

        /** Who the device belongs to, kept after their session ends. */
        val USERNAME = stringPreferencesKey("username")
    }
}
