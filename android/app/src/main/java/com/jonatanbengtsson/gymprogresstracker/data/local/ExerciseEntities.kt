package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.ColumnInfo
import androidx.room.Entity
import androidx.room.PrimaryKey
import kotlin.uuid.Uuid

@Entity(tableName = "exercise")
data class ExerciseEntity(
    @PrimaryKey @ColumnInfo(name = "exercise_id") val exerciseId: Uuid,
    val name: String,
    /** True for an exercise every user can see, false for one of the user's own. */
    @ColumnInfo(name = "is_global") val isGlobal: Boolean
)
