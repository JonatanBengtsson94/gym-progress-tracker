# Code comments

Write comments for someone reading the code for the first time, not for someone reviewing the change that produced it. Don't write comments that explain why you made a change, justify a choice over an alternative, or only make sense if you know what the code looked like before. For example:

- `// Each row pads itself, so a completed set's colour reaches the edges of the card.`
- `// Expands in place like an HTML <details>, without starting the workout.`
- `// Fetched up front so the list is current by the time the user adds an exercise.`
- `// Ids are UUIDs, so the numbers older clients sent are rejected too.`

That reasoning belongs in the commit message.

A comment is still worth writing when it states what something is or does (a KDoc or Go doc comment), or when the code would mislead a reader or break if someone "simplified" it:

- `// The backend stores reps as a uint8, so three digits is the most it can take.`
- `// Keys must be saveable in instance state, which a Uuid isn't.`
- `// Added after the navigation's back handling, so system back closes the picker before the screen.`

When unsure, leave the comment out.

# Database schemas

Neither the backend's database nor the app is deployed anywhere yet, so there is no data to migrate:

- Schema changes go into the existing files in `database/migrations` rather than a new numbered migration. Reset the local database afterwards with `docker compose down -v && docker compose up -d --build`, since migrations only run when it's first created.
- The app's Room database stays at version 1, with no migrations. Change the entities directly and commit the regenerated `android/app/schemas/.../1.json`. Installed builds need their app data cleared.

Backend tests check counts that depend on what the migrations insert, such as the 48 global exercises in `backend/internal/exercise/repository_test.go`, so update them in the same change.

Once either is deployed, this flips: existing migrations and Room versions are never edited, and every change gets a new, data-preserving migration.
