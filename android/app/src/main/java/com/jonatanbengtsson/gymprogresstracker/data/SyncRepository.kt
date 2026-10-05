package com.jonatanbengtsson.gymprogresstracker.data

/** How bringing the device and the server up to date went. */
enum class SyncResult {
    Success,
    /** There's no session, or the server ended it. What wasn't done yet is left for next time. */
    NotLoggedIn,
    NetworkError,
    /** The server rejected or failed part of it. The rest was done. */
    ServerError
}

/** Brings the device and the server up to date with each other. */
interface SyncRepository {
    /**
     * Sends the saved workouts, then replaces the stored templates and exercises with the server's.
     * Stops at a network error or a missing session, leaving the rest for next time.
     */
    suspend fun sync(): SyncResult
}

class DefaultSyncRepository(
    private val workoutsRepository: WorkoutsRepository,
    private val templatesRepository: TemplatesRepository,
    private val exercisesRepository: ExercisesRepository
) : SyncRepository {

    override suspend fun sync(): SyncResult {
        val sent = workoutsRepository.send()
        if (sent == SyncResult.NetworkError || sent == SyncResult.NotLoggedIn) return sent
        var rejected = sent == SyncResult.ServerError
        for (refresh in listOf(templatesRepository::refresh, exercisesRepository::refresh)) {
            when (refresh()) {
                RefreshResult.Success -> {}
                RefreshResult.ServerError -> rejected = true
                RefreshResult.NetworkError -> return SyncResult.NetworkError
                RefreshResult.NotLoggedIn -> return SyncResult.NotLoggedIn
            }
        }
        return if (rejected) SyncResult.ServerError else SyncResult.Success
    }
}
