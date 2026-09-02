# Gym Progress Tracker

## Purpose
The purpose of the application is to provide a easy way to keep track of progression in the gym.

## Local Development

### Requirements
- Docker

### Backend
#### Configure environent variables
Create a .env file at `docker/.env` with the following content:

    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=gym_progress_tracker
    DB_PWD=gym_progress_tracker

#### Running the application
Start the database and backend applications inside docker containers by running `docker compose up -d` inside the `/docker` dir.

### Frontend
TODO
