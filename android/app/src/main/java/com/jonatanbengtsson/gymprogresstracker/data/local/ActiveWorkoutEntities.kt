package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.ColumnInfo
import androidx.room.Embedded
import androidx.room.Entity
import androidx.room.ForeignKey
import androidx.room.PrimaryKey
import androidx.room.Relation

@Entity(tableName = "active_workout_exercise")
data class ActiveWorkoutExerciseEntity(
    @PrimaryKey @ColumnInfo(name = "exercise_id") val exerciseId: Long,
    val name: String,
    /** Where the exercise comes in the workout, counting from 0. */
    val position: Int
)

@Entity(
    tableName = "active_workout_set",
    primaryKeys = ["exercise_id", "position"],
    foreignKeys = [
        ForeignKey(
            entity = ActiveWorkoutExerciseEntity::class,
            parentColumns = ["exercise_id"],
            childColumns = ["exercise_id"],
            onDelete = ForeignKey.CASCADE
        )
    ]
)
data class ActiveWorkoutSetEntity(
    @ColumnInfo(name = "exercise_id") val exerciseId: Long,
    /** Where the set comes in its exercise, counting from 0. */
    val position: Int,
    @ColumnInfo(name = "weight_kg") val weightKg: String,
    val reps: String,
    val completed: Boolean
)

/** [sets] are in no particular order; sort them by [ActiveWorkoutSetEntity.position]. */
data class ActiveWorkoutExerciseWithSets(
    @Embedded val exercise: ActiveWorkoutExerciseEntity,
    @Relation(parentColumn = "exercise_id", entityColumn = "exercise_id")
    val sets: List<ActiveWorkoutSetEntity>
)
