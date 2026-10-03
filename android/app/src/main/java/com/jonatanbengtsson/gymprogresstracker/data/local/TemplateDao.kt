package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.Transaction
import kotlinx.coroutines.flow.Flow

@Dao
interface TemplateDao {

    @Transaction
    @Query("SELECT * FROM template ORDER BY position")
    fun observeTemplates(): Flow<List<TemplateWithSets>>

    @Transaction
    suspend fun replaceTemplates(templates: List<TemplateEntity>, sets: List<TemplateSetEntity>) {
        deleteTemplates()
        insertTemplates(templates)
        insertSets(sets)
    }

    /** Deletes their sets too. */
    @Query("DELETE FROM template")
    suspend fun deleteTemplates()

    @Insert
    suspend fun insertTemplates(templates: List<TemplateEntity>)

    @Insert
    suspend fun insertSets(sets: List<TemplateSetEntity>)
}
