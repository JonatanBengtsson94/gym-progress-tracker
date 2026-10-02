package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.flow.MutableStateFlow

/** Keeps the session in memory only. */
class FakeSessionRepository(session: SessionState = SessionState.LoggedOut) : SessionRepository {

    override val session = MutableStateFlow(session)
    val logIns = mutableListOf<Pair<String, String>>()

    override suspend fun logIn(username: String, sessionId: String) {
        logIns += username to sessionId
        session.value = SessionState.LoggedIn(sessionId)
    }

    override suspend fun endSession(sessionId: String) {
        if (session.value == SessionState.LoggedIn(sessionId)) session.value = SessionState.LoggedOut
    }
}
