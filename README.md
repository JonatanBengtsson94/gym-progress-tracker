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
Create a .env file at `backend/.env` with the following content:

    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=gym_progress_tracker
    DB_PWD=gym_progress_tracker

#### Install dependencies
Install the required dependencies by running `go mod tidy` inside `/backend`

#### Running the application
First start the database by running `docker compose up -d` inside the `/database` dir.
Then start the go application by running `go run .cmd/api`

### Frontend
#### Configure environment variables
Create a .env file at `frontned/.env` with the following content:

    API_URL=
    API_KEY=

#### Install dependencies
Install the required dependencies by running `npm install` inside `/frontend`

#### Running the application
Start the application by running `npm run dev` inside `/frontend`
