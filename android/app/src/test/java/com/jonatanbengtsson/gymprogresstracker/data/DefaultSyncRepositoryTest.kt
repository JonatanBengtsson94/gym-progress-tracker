package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test

class DefaultSyncRepositoryTest {

    private val workoutsRepository = FakeWorkoutsRepository()
    private val templatesRepository = FakeTemplatesRepository()
    private val exercisesRepository = FakeExercisesRepository()
    private val repository = DefaultSyncRepository(workoutsRepository, templatesRepository, exercisesRepository)

    private fun given(sent: SyncResult, templates: RefreshResult = RefreshResult.Success, exercises: RefreshResult = RefreshResult.Success) {
        workoutsRepository.sendResult.complete(sent)
        templatesRepository.refreshResult.complete(templates)
        exercisesRepository.refreshResult.complete(exercises)
    }

    @Test
    fun `sends the workouts, then downloads the templates and exercises`() = runTest {
        given(sent = SyncResult.Success)

        assertEquals(SyncResult.Success, repository.sync())

        assertEquals(1, workoutsRepository.sends)
        assertEquals(1, templatesRepository.refreshes)
        assertEquals(1, exercisesRepository.refreshes)
    }

    @Test
    fun `a rejected workout still downloads the rest and reports a server error`() = runTest {
        given(sent = SyncResult.ServerError)

        assertEquals(SyncResult.ServerError, repository.sync())

        assertEquals(1, templatesRepository.refreshes)
        assertEquals(1, exercisesRepository.refreshes)
    }

    @Test
    fun `a network error sending stops before downloading`() = runTest {
        assertStopsAfterSending(SyncResult.NetworkError)
    }

    @Test
    fun `no session sending stops before downloading`() = runTest {
        assertStopsAfterSending(SyncResult.NotLoggedIn)
    }

    @Test
    fun `a server error downloading the templates still downloads the exercises`() = runTest {
        given(sent = SyncResult.Success, templates = RefreshResult.ServerError)

        assertEquals(SyncResult.ServerError, repository.sync())

        assertEquals(1, exercisesRepository.refreshes)
    }

    @Test
    fun `a network error downloading the templates stops the sync`() = runTest {
        given(sent = SyncResult.Success, templates = RefreshResult.NetworkError)

        assertEquals(SyncResult.NetworkError, repository.sync())

        assertEquals(0, exercisesRepository.refreshes)
    }

    @Test
    fun `no session downloading the templates stops the sync`() = runTest {
        given(sent = SyncResult.Success, templates = RefreshResult.NotLoggedIn)

        assertEquals(SyncResult.NotLoggedIn, repository.sync())

        assertEquals(0, exercisesRepository.refreshes)
    }

    @Test
    fun `a network error downloading the exercises is reported`() = runTest {
        given(sent = SyncResult.Success, exercises = RefreshResult.NetworkError)

        assertEquals(SyncResult.NetworkError, repository.sync())
    }

    @Test
    fun `a server error downloading the exercises is reported`() = runTest {
        given(sent = SyncResult.Success, exercises = RefreshResult.ServerError)

        assertEquals(SyncResult.ServerError, repository.sync())
    }

    @Test
    fun `no session downloading the exercises is reported`() = runTest {
        given(sent = SyncResult.Success, exercises = RefreshResult.NotLoggedIn)

        assertEquals(SyncResult.NotLoggedIn, repository.sync())
    }

    private suspend fun assertStopsAfterSending(result: SyncResult) {
        given(sent = result)

        assertEquals(result, repository.sync())

        assertEquals(0, templatesRepository.refreshes)
        assertEquals(0, exercisesRepository.refreshes)
    }
}
