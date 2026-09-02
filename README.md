# Gym Progress Tracker

## Purpose
The purpose of the application is to provide a easy way to keep track of progression in the gym.

## Local Development

### Requirements
- Go 1.26+
- Node.js 26+
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
#### Configure environment variables
Create a .env file at `frontned/.env` with the following content:

    API_URL=
    API_KEY=

#### Install dependencies
Install the required dependencies by running `npm install` inside `/frontend`

#### Running the application
Start the application by running `npm run dev` inside `/frontend`
