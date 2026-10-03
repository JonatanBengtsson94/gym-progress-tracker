package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.Transaction
import kotlinx.coroutines.flow.Flow

@Dao
interface ExerciseDao {

    @Query("SELECT * FROM exercise ORDER BY position")
    fun observeExercises(): Flow<List<ExerciseEntity>>

    @Transaction
    suspend fun replaceExercises(exercises: List<ExerciseEntity>) {
        deleteExercises()
        insertExercises(exercises)
    }

    @Query("DELETE FROM exercise")
    suspend fun deleteExercises()

    @Insert
    suspend fun insertExercises(exercises: List<ExerciseEntity>)
}
