package com.jonatanbengtsson.gymprogresstracker.data

/** How replacing the data stored on the device with the server's went. */
enum class RefreshResult {
    Success,
    /** The session had ended, which logs the user out. */
    SessionExpired,
    NetworkError,
    ServerError
}
