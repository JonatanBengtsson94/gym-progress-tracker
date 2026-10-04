package com.jonatanbengtsson.gymprogresstracker.data.local

import androidx.room.AutoMigration
import androidx.room.Database
import androidx.room.RoomDatabase
import androidx.room.TypeConverters

@Database(
    entities = [
        ActiveWorkoutEntity::class,
        ActiveWorkoutExerciseEntity::class,
        ActiveWorkoutSetEntity::class,
        ExerciseEntity::class,
        TemplateEntity::class,
        TemplateSetEntity::class
    ],
    version = 3,
    autoMigrations = [AutoMigration(from = 2, to = 3)]
)
@TypeConverters(Converters::class)
abstract class GymDatabase : RoomDatabase() {
    abstract fun activeWorkoutDao(): ActiveWorkoutDao
    abstract fun exerciseDao(): ExerciseDao
    abstract fun templateDao(): TemplateDao
}
