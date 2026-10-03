package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.ColumnInfo
import androidx.room.Entity
import androidx.room.PrimaryKey
import kotlin.uuid.Uuid

@Entity(tableName = "exercise")
data class ExerciseEntity(
    @PrimaryKey @ColumnInfo(name = "exercise_id") val exerciseId: Uuid,
    val name: String,
    /** Where the server listed the exercise, counting from 0. */
    val position: Int
)
