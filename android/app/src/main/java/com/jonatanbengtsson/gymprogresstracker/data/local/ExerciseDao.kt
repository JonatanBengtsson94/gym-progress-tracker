package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.Transaction
import kotlinx.coroutines.flow.Flow

@Dao
interface ExerciseDao {

    @Query("SELECT * FROM exercise")
    fun observeExercises(): Flow<List<ExerciseEntity>>

    /** Replaces the global exercises, or the user's own, with [exercises], keeping the other kind. */
    @Transaction
    suspend fun replaceExercises(isGlobal: Boolean, exercises: List<ExerciseEntity>) {
        deleteExercises(isGlobal)
        insertExercises(exercises)
    }

    @Query("DELETE FROM exercise WHERE is_global = :isGlobal")
    suspend fun deleteExercises(isGlobal: Boolean)

    @Insert
    suspend fun insertExercises(exercises: List<ExerciseEntity>)
}
