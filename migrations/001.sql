CREATE TABLE users (
  user_id SERIAL PRIMARY KEY,
  username VARCHAR(100) NOT NULL,
  password VARCHAR(255) NOT NULL,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  date_of_birth DATE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE templates (
  template_id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(user_id),
  template_name VARCHAR(100) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE workouts (
  workout_id SERIAL PRIMARY KEY,
  template_id INTEGER REFERENCES templates(template_id),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE exercises (
  exercise_id SERIAL PRIMARY KEY,
  exercise_name VARCHAR(100) NOT NULL
);

CREATE TABLE sets (
  set_id SERIAL PRIMARY KEY,
  exercise_id INTEGER NOT NULL REFERENCES exercises(exercise_id),
  workout_id INTEGER NOT NULL REFERENCES workouts(workout_id),
  reps INTEGER NOT NULL,
  weight NUMERIC(6,2) NOT NULL
);
