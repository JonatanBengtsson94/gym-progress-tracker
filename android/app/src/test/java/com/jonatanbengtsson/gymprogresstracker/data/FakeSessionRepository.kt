package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.flow.MutableStateFlow

/** Keeps the session in memory only. */
class FakeSessionRepository(session: SessionState = SessionState.LoggedOut(null)) : SessionRepository {

    override val session = MutableStateFlow(session)
    val logIns = mutableListOf<Pair<String, String>>()

    override suspend fun logIn(username: String, sessionId: String) {
        logIns += username to sessionId
        session.value = SessionState.LoggedIn(sessionId, username)
    }

    override suspend fun endSession(sessionId: String) {
        val current = session.value
        if (current is SessionState.LoggedIn && current.sessionId == sessionId) session.value = SessionState.LoggedOut(current.username)
    }
}
