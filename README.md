# Gym Progress Tracker

An easy way to keep track of progression in the gym.

## Local Development

### Backend

#### Requirements
- Docker
- Go 1.27 (only for running the tests; the backend itself runs in Docker)

#### Setup
Create `docker/.env`:

    DB_USER=gym
    DB_PWD=gym
    DB_NAME=gym
    SESSION_DURATION=24h

Then, from the `docker/` directory:

    docker compose up -d --build

| Service  | URL / port                                        |
|----------|---------------------------------------------------|
| Backend  | `localhost:8080`                                  |
| Postgres | `localhost:5432`                                  |
| pgAdmin  | `localhost:5050` (`admin@example.com` / `admin`)  |

These credentials are for local development only.

#### Create a user
There is intentionally no registration endpoint. Create users with the `createuser` command, which stores only a bcrypt hash of the password (from `docker/`):

    docker compose exec backend go run ./cmd/createuser -username alice -first-name Alice -last-name Anderson

You will be prompted for the password twice. When stdin is not a terminal, the password is read from the first line of input instead:

    echo 'secret' | docker compose exec -T backend go run ./cmd/createuser -username alice -first-name Alice -last-name Anderson

Then log in and use the returned `session_id`:

    curl -X POST localhost:8080/login -d '{"username":"alice","password":"secret"}'
    curl localhost:8080/workouts -H "Authorization: Bearer <session_id>"

#### Database migrations
Files in `database/migrations` run **only when the database is first created**. After changing or adding a migration, reset the database (this deletes all data):

    docker compose down -v && docker compose up -d --build

#### Tests
From `backend/`, with Docker running (tests start their own Postgres container):

    go test ./...

### Android app
The app lives in `android/` and needs a device or emulator running Android 16 (API 36) or newer.

#### Requirements
- Android Studio, or JDK 25 and the Android SDK

#### Backend URL
Debug builds talk to `http://10.0.2.2:8080`, which is the host machine as seen from the emulator. To use a physical device, point it at your machine's LAN address in `android/local.properties`:

    api.baseUrl=http://192.168.1.10:8080

Plain HTTP is only allowed in debug builds.

#### Run
Open `android/` in Android Studio and run the `app` configuration, or from `android/`:

    ./gradlew installDebug

#### Tests
From `android/`, unit tests:

    ./gradlew testDebugUnitTest

UI tests, on a connected device or a running emulator:

    ./gradlew connectedDebugAndroidTest
