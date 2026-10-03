package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.ColumnInfo
import androidx.room.Embedded
import androidx.room.Entity
import androidx.room.ForeignKey
import androidx.room.PrimaryKey
import androidx.room.Relation
import java.time.Instant
import kotlin.uuid.Uuid

@Entity(tableName = "template")
data class TemplateEntity(
    @PrimaryKey @ColumnInfo(name = "template_id") val templateId: Uuid,
    val name: String,
    /** Where the server listed the template, counting from 0. */
    val position: Int,
    /** Null when no workout has been logged under the template yet. */
    @Embedded(prefix = "latest_") val latestWorkout: TemplateLatestWorkout?
)

data class TemplateLatestWorkout(
    @ColumnInfo(name = "workout_id") val workoutId: Uuid,
    @ColumnInfo(name = "started_at") val startedAt: Instant,
    @ColumnInfo(name = "completed_at") val completedAt: Instant
)

/** A set of a template's latest workout. */
@Entity(
    tableName = "template_set",
    primaryKeys = ["template_id", "position"],
    foreignKeys = [
        ForeignKey(
            entity = TemplateEntity::class,
            parentColumns = ["template_id"],
            childColumns = ["template_id"],
            onDelete = ForeignKey.CASCADE
        )
    ]
)
data class TemplateSetEntity(
    @ColumnInfo(name = "template_id") val templateId: Uuid,
    /** Where the set comes in the workout, counting from 0. */
    val position: Int,
    @ColumnInfo(name = "exercise_id") val exerciseId: Uuid,
    @ColumnInfo(name = "exercise_name") val exerciseName: String,
    val reps: Int,
    @ColumnInfo(name = "weight_grams") val weightGrams: Int
)

/** [sets] are in no particular order; sort them by [TemplateSetEntity.position]. */
data class TemplateWithSets(
    @Embedded val template: TemplateEntity,
    @Relation(parentColumn = "template_id", entityColumn = "template_id")
    val sets: List<TemplateSetEntity>
)
