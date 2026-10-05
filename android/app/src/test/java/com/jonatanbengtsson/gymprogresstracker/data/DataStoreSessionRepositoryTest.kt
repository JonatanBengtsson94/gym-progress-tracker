package com.jonatanbengtsson.gymprogresstracker.data

import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

class DataStoreSessionRepositoryTest {

    @get:Rule
    val folder = TemporaryFolder()

    private val file by lazy { File(folder.root, "session.preferences_pb") }

    /** A repository over the same file each time, as after the app restarts. Cancel [scope] before the next one. */
    private fun repository(scope: CoroutineScope) = DataStoreSessionRepository(PreferenceDataStoreFactory.create(scope = scope) { file }, scope)

    private fun TestScope.newRunScope() = CoroutineScope(coroutineContext + Job())

    private suspend fun SessionRepository.loadedSession() = session.first { it != SessionState.Loading }

    private suspend fun SessionRepository.awaitSession(expected: SessionState) = session.first { it == expected }

    @Test
    fun `nobody is logged in at first`() = runTest {
        val repository = repository(backgroundScope)

        assertEquals(SessionState.Loading, repository.session.value)
        assertEquals(SessionState.LoggedOut(null), repository.loadedSession())
    }

    @Test
    fun `logging in starts a session`() = runTest {
        val repository = repository(backgroundScope)

        repository.logIn("alice", "session-123")

        assertEquals(SessionState.LoggedIn("session-123", "alice"), repository.awaitSession(SessionState.LoggedIn("session-123", "alice")))
    }

    @Test
    fun `a session survives a restart`() = runTest {
        val firstRun = newRunScope()
        repository(firstRun).logIn("alice", "session-123")
        firstRun.cancel()

        val repository = repository(backgroundScope)

        assertEquals(SessionState.LoggedIn("session-123", "alice"), repository.loadedSession())
    }

    @Test
    fun `ending the session logs out but remembers who the device belongs to`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-123")

        repository.endSession("session-123")

        assertEquals(SessionState.LoggedOut("alice"), repository.awaitSession(SessionState.LoggedOut("alice")))
    }

    @Test
    fun `who the device belongs to survives a restart`() = runTest {
        val firstRun = newRunScope()
        repository(firstRun).apply {
            logIn("alice", "session-123")
            endSession("session-123")
        }
        firstRun.cancel()

        val repository = repository(backgroundScope)

        assertEquals(SessionState.LoggedOut("alice"), repository.loadedSession())
    }

    @Test
    fun `ending a session that has been replaced is ignored`() = runTest {
        val repository = repository(backgroundScope)
        repository.logIn("alice", "session-1")
        repository.logIn("alice", "session-2")

        repository.endSession("session-1")

        assertEquals(SessionState.LoggedIn("session-2", "alice"), repository.awaitSession(SessionState.LoggedIn("session-2", "alice")))
    }
}
