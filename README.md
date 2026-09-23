# Gym Progress Tracker

An easy way to keep track of progression in the gym.

## Local Development

### Requirements
- Docker
- Go 1.27

### Backend
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

### Frontend
TODO
