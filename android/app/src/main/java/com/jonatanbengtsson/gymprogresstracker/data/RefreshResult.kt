package com.jonatanbengtsson.gymprogresstracker.data

/** How replacing the data stored on the device with the server's went. */
enum class RefreshResult {
    Success,
    /** There's no session, or the server ended it. Nothing was replaced. */
    NotLoggedIn,
    NetworkError,
    ServerError
}
