package com.jonatanbengtsson.gymprogresstracker

import android.app.Application

class GymProgressTrackerApplication : Application() {

    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
    }
}
