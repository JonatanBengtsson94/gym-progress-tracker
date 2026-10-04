package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateDao
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateLatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateSetEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateWithSets
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.combine
import java.time.Instant

/** The user's templates, each with its latest workout, kept on the device so they're there offline too. */
interface TemplatesRepository {
    /**
     * Most recently performed first, including the workouts saved but not yet synced and the templates
     * they create. Holds only those until the templates have been fetched once.
     */
    val templates: Flow<List<WorkoutTemplate>>

    /** Replaces the stored templates with the server's. They're kept as they are if that fails. */
    suspend fun refresh(): RefreshResult
}

class RoomTemplatesRepository(
    private val api: TemplatesApi,
    private val dao: TemplateDao,
    pendingWorkoutsRepository: PendingWorkoutsRepository
) : TemplatesRepository {

    override val templates: Flow<List<WorkoutTemplate>> =
        combine(dao.observeTemplates(), pendingWorkoutsRepository.workouts) { templates, pending ->
            withPendingWorkouts(templates.map { it.toTemplate() }, pending)
        }

    override suspend fun refresh(): RefreshResult = when (val result = api.getTemplates()) {
        is ApiResult.Success -> {
            dao.replaceTemplates(
                templates = result.value.mapIndexed { position, template -> template.toEntity(position) },
                sets = result.value.flatMap { it.toSetEntities() }
            )
            RefreshResult.Success
        }
        ApiResult.Unauthorized -> RefreshResult.SessionExpired
        ApiResult.NetworkError -> RefreshResult.NetworkError
        ApiResult.ServerError -> RefreshResult.ServerError
    }

    /**
     * [templates] with each of the [pending] workouts as its template's latest workout when it's newer,
     * adding the templates they create, re-sorted most recently performed first. A template is matched
     * by id, or by name ignoring case, since the server returns its own template for a name it already
     * has when the one created on the device is synced.
     */
    private fun withPendingWorkouts(templates: List<WorkoutTemplate>, pending: List<PendingWorkout>): List<WorkoutTemplate> {
        val merged = templates.toMutableList()
        for ((workoutId, templateId, _, workout) in pending) {
            val latest = LatestWorkout(workoutId, workout.startedAt, workout.completedAt, workout.exercises)
            val index = merged.indexOfFirst { it.id == templateId }
                .takeIf { it >= 0 } ?: merged.indexOfFirst { it.name.equals(workout.name, ignoreCase = true) }
            if (index < 0) {
                merged += WorkoutTemplate(templateId, workout.name, latest)
            } else if ((merged[index].latestWorkout?.completedAt ?: Instant.MIN) < workout.completedAt) {
                merged[index] = merged[index].copy(latestWorkout = latest)
            }
        }
        // Stable, so templates never performed keep the server's order.
        return merged.sortedWith(compareByDescending(nullsFirst()) { it.latestWorkout?.completedAt })
    }

    private fun WorkoutTemplate.toEntity(position: Int) = TemplateEntity(
        templateId = id,
        name = name,
        position = position,
        latestWorkout = latestWorkout?.let { TemplateLatestWorkout(it.workoutId, it.startedAt, it.completedAt) }
    )

    /** The latest workout's sets in the order they were logged, which is grouped by exercise. */
    private fun WorkoutTemplate.toSetEntities(): List<TemplateSetEntity> =
        latestWorkout?.exercises.orEmpty()
            .flatMap { exercise -> exercise.sets.map { set -> exercise to set } }
            .mapIndexed { position, (exercise, set) ->
                TemplateSetEntity(id, position, exercise.exerciseId, exercise.name, set.reps, set.weightGrams)
            }

    private fun TemplateWithSets.toTemplate() = WorkoutTemplate(
        id = template.templateId,
        name = template.name,
        latestWorkout = template.latestWorkout?.let { latest ->
            LatestWorkout(
                workoutId = latest.workoutId,
                startedAt = latest.startedAt,
                completedAt = latest.completedAt,
                exercises = sets.sortedBy { it.position }
                    .groupBy { it.exerciseId }
                    .map { (exerciseId, sets) ->
                        WorkoutExercise(exerciseId, sets.first().exerciseName, sets.map { WorkoutSet(it.reps, it.weightGrams) })
                    }
            )
        }
    )
}
