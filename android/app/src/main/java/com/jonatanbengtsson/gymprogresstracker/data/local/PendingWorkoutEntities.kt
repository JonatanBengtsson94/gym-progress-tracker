package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.ColumnInfo
import androidx.room.Embedded
import androidx.room.Entity
import androidx.room.ForeignKey
import androidx.room.PrimaryKey
import androidx.room.Relation
import java.time.Instant
import kotlin.uuid.Uuid

/** A finished workout waiting to be sent to the server. */
@Entity(tableName = "pending_workout")
data class PendingWorkoutEntity(
    @PrimaryKey @ColumnInfo(name = "workout_id") val workoutId: Uuid,
    @ColumnInfo(name = "template_id") val templateId: Uuid,
    /** True when the template was created on the device and may not be on the server yet. */
    @ColumnInfo(name = "template_is_new") val templateIsNew: Boolean,
    val name: String,
    @ColumnInfo(name = "started_at") val startedAt: Instant,
    @ColumnInfo(name = "completed_at") val completedAt: Instant
)

@Entity(
    tableName = "pending_workout_set",
    primaryKeys = ["workout_id", "position"],
    foreignKeys = [
        ForeignKey(
            entity = PendingWorkoutEntity::class,
            parentColumns = ["workout_id"],
            childColumns = ["workout_id"],
            onDelete = ForeignKey.CASCADE
        )
    ]
)
data class PendingWorkoutSetEntity(
    @ColumnInfo(name = "workout_id") val workoutId: Uuid,
    /** Where the set comes in the workout, counting from 0. */
    val position: Int,
    @ColumnInfo(name = "exercise_id") val exerciseId: Uuid,
    @ColumnInfo(name = "exercise_name") val exerciseName: String,
    val reps: Int,
    @ColumnInfo(name = "weight_grams") val weightGrams: Int
)

/** [sets] are in no particular order; sort them by [PendingWorkoutSetEntity.position]. */
data class PendingWorkoutWithSets(
    @Embedded val workout: PendingWorkoutEntity,
    @Relation(parentColumn = "workout_id", entityColumn = "workout_id")
    val sets: List<PendingWorkoutSetEntity>
)
