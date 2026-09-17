-- USERS

CREATE TABLE users (
  user_id SERIAL PRIMARY KEY,
  username VARCHAR(100) UNIQUE NOT NULL,
  password VARCHAR(255) NOT NULL,
  first_name VARCHAR(100) NOT NULL,
  last_name VARCHAR(100) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- TEMPLATES

CREATE TABLE templates (
  template_id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(user_id),
  template_name VARCHAR(100) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uq_templates_user_name
  ON templates(user_id, lower(template_name));

CREATE INDEX idx_templates_user_id
  ON templates(user_id);

-- WORKOUTS

CREATE TABLE workouts (
  workout_id SERIAL PRIMARY KEY,
  template_id INTEGER NOT NULL REFERENCES templates(template_id),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP
);

CREATE INDEX idx_workouts_template_id 
  ON workouts(template_id);

-- EXERCISES

CREATE TABLE exercises (
  exercise_id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(user_id),
  exercise_name VARCHAR(100) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uq_exercises_user_name
  ON exercises (user_id, lower(exercise_name))
  WHERE user_id IS NOT NULL;

CREATE UNIQUE INDEX uq_exercises_global_name 
  ON exercises (lower(exercise_name)) 
  WHERE user_id IS NULL;

-- SETS

CREATE TABLE sets (
  set_id SERIAL PRIMARY KEY,
  exercise_id INTEGER NOT NULL REFERENCES exercises(exercise_id),
  workout_id INTEGER NOT NULL REFERENCES workouts(workout_id),
  reps INTEGER NOT NULL,
  weight_grams INTEGER NOT NULL
);

CREATE INDEX idx_sets_workout_id 
  ON sets(workout_id);
CREATE INDEX idx_sets_exercise_id 
  ON sets(exercise_id);

