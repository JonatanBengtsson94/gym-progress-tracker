package com.jonatanbengtsson.gymprogresstracker.ui.user

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.jonatanbengtsson.gymprogresstracker.R
import com.jonatanbengtsson.gymprogresstracker.ui.theme.GymProgressTrackerTheme

@Composable
fun UserScreen(
    onLogIn: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: UserViewModel = viewModel(factory = UserViewModel.Factory)
) {
    LaunchedEffect(viewModel.uiState.logInRequested) {
        if (viewModel.uiState.logInRequested) {
            viewModel.onLogInShown()
            onLogIn()
        }
    }

    UserContent(
        uiState = viewModel.uiState,
        onSync = viewModel::sync,
        modifier = modifier
    )
}

@Composable
fun UserContent(
    uiState: UserUiState,
    onSync: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        if (uiState.username == null) {
            Text(text = stringResource(R.string.user_not_logged_in), style = MaterialTheme.typography.headlineMedium)
        } else {
            Column {
                Text(
                    text = stringResource(if (uiState.isLoggedIn) R.string.user_logged_in_as else R.string.user_logged_out),
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(text = uiState.username, style = MaterialTheme.typography.headlineMedium)
            }
        }
        HorizontalDivider()
        SyncSection(
            pendingWorkouts = uiState.pendingWorkouts,
            isSyncing = uiState.isSyncing,
            errorMessage = uiState.syncErrorMessage,
            onSync = onSync
        )
    }
}

@Composable
private fun SyncSection(pendingWorkouts: Int, isSyncing: Boolean, @StringRes errorMessage: Int?, onSync: () -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            text = if (pendingWorkouts > 0) {
                pluralStringResource(R.plurals.user_pending, pendingWorkouts, pendingWorkouts)
            } else {
                stringResource(R.string.user_all_synced)
            },
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Button(
            onClick = onSync,
            enabled = !isSyncing,
            modifier = Modifier.fillMaxWidth()
        ) {
            if (isSyncing) {
                CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(stringResource(R.string.user_sync))
            }
        }
        if (errorMessage != null) {
            Text(
                text = stringResource(errorMessage),
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodyMedium
            )
        }
    }
}

@Preview(showBackground = true)
@Composable
fun UserContentPreview() {
    GymProgressTrackerTheme {
        UserContent(uiState = UserUiState(username = "alice", isLoggedIn = true, pendingWorkouts = 2), onSync = {})
    }
}

@Preview(showBackground = true)
@Composable
fun UserContentNotLoggedInPreview() {
    GymProgressTrackerTheme {
        UserContent(uiState = UserUiState(pendingWorkouts = 1), onSync = {})
    }
}

@Preview(showBackground = true)
@Composable
fun UserContentLoggedOutPreview() {
    GymProgressTrackerTheme {
        UserContent(uiState = UserUiState(username = "alice", pendingWorkouts = 1), onSync = {})
    }
}
