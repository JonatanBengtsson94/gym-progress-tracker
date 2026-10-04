package com.jonatanbengtsson.gymprogresstracker.data

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlin.uuid.Uuid

/** Logs finished workouts on the server. */
interface WorkoutsRepository {
    /**
     * Saves [workout] as the user's workout [workoutId], replacing it if it was saved before. A workout
     * without a template id is logged under the user's template with the same name, or a new one if
     * there is none.
     */
    suspend fun save(workoutId: Uuid, workout: FinishedWorkout): ApiResult<Unit>
}

/** Refreshes the templates in [externalScope] after each save, so they show the new workout. */
class ApiWorkoutsRepository(
    private val api: WorkoutsApi,
    private val templatesRepository: TemplatesRepository,
    private val externalScope: CoroutineScope
) : WorkoutsRepository {

    override suspend fun save(workoutId: Uuid, workout: FinishedWorkout): ApiResult<Unit> {
        // Template names are unique per user regardless of case, so the server rejects a new template
        // whose name only differs in case from an existing one.
        val templateId = workout.templateId ?: templatesRepository.templates.first()
            .find { it.name.equals(workout.templateName.trim(), ignoreCase = true) }?.id
        val result = api.putWorkout(workoutId, workout.copy(templateId = templateId))
        if (result is ApiResult.Success) externalScope.launch { templatesRepository.refresh() }
        return result
    }
}
