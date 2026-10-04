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
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import java.io.IOException

sealed interface SessionState {
    /** The saved session hasn't been read from disk yet. */
    data object Loading : SessionState
    data object LoggedOut : SessionState
    data class LoggedIn(val sessionId: String) : SessionState
}

/** The session the user is logged in with, kept on the device so they stay logged in across restarts. */
interface SessionRepository {
    val session: StateFlow<SessionState>

    /**
     * Starts [sessionId] for [username]. If someone else logged in last, the workout they left in
     * progress and the workouts they saved but didn't sync are thrown away first, so the new user
     * never sees them and they're never sent under the new user's account.
     */
    suspend fun logIn(username: String, sessionId: String)

    /**
     * Ends [sessionId], when the user logs out or the server rejects it. Ignored if another session
     * has started since. The workout in progress is kept for when the same user logs in again.
     */
    suspend fun endSession(sessionId: String)

    /** Ends the current session, if there is one, keeping the workout in progress like [endSession]. */
    suspend fun logOut()
}

class DataStoreSessionRepository(
    private val dataStore: DataStore<Preferences>,
    private val activeWorkoutRepository: ActiveWorkoutRepository,
    private val pendingWorkoutsRepository: PendingWorkoutsRepository,
    externalScope: CoroutineScope
) : SessionRepository {

    // An unreadable file is treated as empty, which logs the user out rather than crashing.
    private val preferences = dataStore.data.catch { if (it is IOException) emit(emptyPreferences()) else throw it }

    override val session: StateFlow<SessionState> = preferences
        .map { preferences -> preferences[SESSION_ID]?.let(SessionState::LoggedIn) ?: SessionState.LoggedOut }
        .stateIn(externalScope, SharingStarted.Eagerly, SessionState.Loading)

    override suspend fun logIn(username: String, sessionId: String) {
        if (preferences.first()[USERNAME] != username) {
            activeWorkoutRepository.workout.filterNotNull().first()
            activeWorkoutRepository.update { ActiveWorkout() }
            pendingWorkoutsRepository.clear()
        }
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

    override suspend fun logOut() {
        dataStore.edit { preferences -> preferences.remove(SESSION_ID) }
    }

    private companion object {
        val SESSION_ID = stringPreferencesKey("session_id")

        /** Who logged in last, kept after their session ends. */
        val USERNAME = stringPreferencesKey("username")
    }
}
