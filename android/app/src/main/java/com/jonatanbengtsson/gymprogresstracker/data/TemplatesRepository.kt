package com.jonatanbengtsson.gymprogresstracker.data

import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateDao
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateLatestWorkout
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateSetEntity
import com.jonatanbengtsson.gymprogresstracker.data.local.TemplateWithSets
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

/** The user's templates, each with its latest workout, kept on the device so they're there offline too. */
interface TemplatesRepository {
    /** Most recently performed first. Empty until they've been fetched once. */
    val templates: Flow<List<WorkoutTemplate>>

    /** Replaces the stored templates with the server's. They're kept as they are if that fails. */
    suspend fun refresh(sessionId: String): RefreshResult
}

class RoomTemplatesRepository(private val api: TemplatesApi, private val dao: TemplateDao) : TemplatesRepository {

    override val templates: Flow<List<WorkoutTemplate>> =
        dao.observeTemplates().map { templates -> templates.map { it.toTemplate() } }

    override suspend fun refresh(sessionId: String): RefreshResult = when (val result = api.getTemplates(sessionId)) {
        is TemplatesResult.Success -> {
            dao.replaceTemplates(
                templates = result.templates.mapIndexed { position, template -> template.toEntity(position) },
                sets = result.templates.flatMap { it.toSetEntities() }
            )
            RefreshResult.Success
        }
        TemplatesResult.SessionExpired -> RefreshResult.SessionExpired
        TemplatesResult.NetworkError -> RefreshResult.NetworkError
        TemplatesResult.ServerError -> RefreshResult.ServerError
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
