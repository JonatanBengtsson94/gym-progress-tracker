package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlin.uuid.Uuid

/** Finished workouts, saved on the device and sent to the server by [send]. */
interface WorkoutsRepository {
    /** How many saved workouts haven't been sent to the server yet. */
    val pendingCount: Flow<Int>

    /**
     * Saves [workout] on the device as workout [workoutId], replacing one saved before under the same
     * id. It's logged under the user's template with the workout's name, ignoring case, or under a new
     * template created on the device if there is none.
     */
    suspend fun save(workoutId: Uuid, workout: FinishedWorkout)

    /**
     * Sends the saved workouts to the server, oldest first, creating their templates first where needed,
     * and removes each one the server accepts. Stops at the first network error, leaving the rest for
     * next time. [SyncResult.ServerError] means the server rejected at least one; the others were sent.
     */
    suspend fun send(): SyncResult
}

class ApiWorkoutsRepository(
    private val workoutsApi: WorkoutsApi,
    private val templatesApi: TemplatesApi,
    private val pendingWorkoutsRepository: PendingWorkoutsRepository,
    private val templatesRepository: TemplatesRepository
) : WorkoutsRepository {

    override val pendingCount: Flow<Int> = pendingWorkoutsRepository.workouts.map { it.size }

    override suspend fun save(workoutId: Uuid, workout: FinishedWorkout) {
        // Template names are unique per user regardless of case, so the server rejects a new template
        // whose name only differs in case from an existing one.
        val existing = templatesRepository.templates.first().find { it.name.equals(workout.name, ignoreCase = true) }?.id
        // A template matched here may itself have been created on the device by a workout still waiting to sync.
        val templateIsNew = existing == null ||
            pendingWorkoutsRepository.workouts.first().any { it.templateIsNew && it.templateId == existing }
        pendingWorkoutsRepository.add(PendingWorkout(workoutId, existing ?: Uuid.random(), templateIsNew, workout))
    }

    override suspend fun send(): SyncResult {
        var rejected = false
        for (pending in pendingWorkoutsRepository.workouts.first()) {
            val templateId = if (pending.templateIsNew) {
                when (val created = templatesApi.createTemplate(pending.templateId, pending.workout.name)) {
                    is ApiResult.Success -> created.value
                    ApiResult.ServerError -> {
                        rejected = true
                        continue
                    }
                    ApiResult.NetworkError -> return SyncResult.NetworkError
                    ApiResult.Unauthorized -> return SyncResult.NotLoggedIn
                }
            } else {
                pending.templateId
            }
            when (workoutsApi.putWorkout(pending.workoutId, templateId, pending.workout)) {
                is ApiResult.Success -> pendingWorkoutsRepository.remove(pending.workoutId)
                ApiResult.ServerError -> rejected = true
                ApiResult.NetworkError -> return SyncResult.NetworkError
                ApiResult.Unauthorized -> return SyncResult.NotLoggedIn
            }
        }
        return if (rejected) SyncResult.ServerError else SyncResult.Success
    }
}
